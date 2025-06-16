package telegram_bot

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	telegramAuthPlugin "github.com/docker-pet/backend/plugins/telegram_auth"
	tele "gopkg.in/telebot.v4"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type Options struct {
  AuthPlugin *telegramAuthPlugin.Plugin
  CronExpr   string
  CronUsersPerRun int
  CronUserSyncInterval time.Duration
}

type Plugin struct {
	app        core.App
	options    *Options
  appConfig  *core.Record
}

// Validate plugin options.
func (p *Plugin) Validate() error {
	if p.options == nil {
		return fmt.Errorf("options is required")
	}

  if p.options.AuthPlugin == nil {
    return fmt.Errorf("AuthPlugin is required")
  }

  if p.options.CronExpr == "" {
    return fmt.Errorf("CronExpr is required")
  }

  if p.options.CronUsersPerRun <= 0 {
    return fmt.Errorf("CronUsersPerRun must be greater than 0")
  }

	return nil
}

// Register the register plugin and panic if error occurred
func Register(app core.App, options *Options) *Plugin {
	if p, err := RegisterWrapper(app, options); err != nil {
		panic(err)
	} else {
		return p
	}
}

// Plugin registration
func RegisterWrapper(app core.App, options *Options) (*Plugin, error) {

  // Create plugin options
	p := &Plugin{
		app:        app,
    options:    options,
	}

  // Start the bot
  app.OnServe().BindFunc(func(e *core.ServeEvent) error {
    appConfig, err := app.FindFirstRecordByFilter("app", "id != ''")
    if err != nil {
      return fmt.Errorf("failed to find application configuration: %w", err)
    }

    p.appConfig = appConfig
    p.StartBot(e)
    return e.Next()
  })

	return p, nil
}

func (p *Plugin) StartBot(e *core.ServeEvent) {
  p.app.Logger().Info("Telegram bot starting")
  botToken := p.appConfig.GetString("telegramBotToken")
  appDomain := p.appConfig.GetString("appDomain")
  endpointPath := fmt.Sprintf("/api/telegram_bot/%s", botToken)

  webhook := &tele.Webhook{
    Endpoint: &tele.WebhookEndpoint{
      PublicURL: "https://" + appDomain + endpointPath,
    },

    Listen: "",
  
    AllowedUpdates: []string{
      "my_chat_member",
      "chat_member",
      "chat_join_request",
      "message",
    },
  }

  bot, err := tele.NewBot(tele.Settings{
    Token: botToken,
    Poller: webhook,
  })

  if err != nil {
    p.app.Logger().Error("Telegram bot init error:", "Err", err)
    return
  }

  // Webhook Pooler
  _, err = bot.Webhook()
  if err != nil {
    p.app.Logger().Error("Telegram bot Webhook init error:", "Err", err)
    return
  }

  // HTTP Endpoint
  e.Router.POST(endpointPath, func(ctx *core.RequestEvent) error {
    webhook.ServeHTTP(ctx.Response, ctx.Request)
    return ctx.JSON(http.StatusOK, map[string]bool{"ok": true})
  })

  // Middlewares
  // bot.Use(middleware.Logger())
  bot.Use(func (next tele.HandlerFunc) tele.HandlerFunc {
    return func(c tele.Context) error {
      // User & Bot chat
      if c.Chat().Type == tele.ChatPrivate {
        return next(c)
      }

      // Unauthorized chat check
      if c.Chat().ID != int64(p.appConfig.GetInt("telegramChannelId")) && c.Chat().ID != int64(p.appConfig.GetInt("telegramPremiumChannelId")) {
        p.app.Logger().Info(
          "Telegram bot received message from unauthorized chat.",
          "ChatId", c.Chat().ID,
          "Title", c.Chat().Title,
        )

        c.Send(
          "This bot is not authorized to work in this chat (<code>" + fmt.Sprint(c.Chat().ID) + "</code>).",
          &tele.SendOptions{
            ParseMode: tele.ModeHTML,
            DisableNotification: true,
          },
        )

        return c.Bot().Leave(c.Chat())
      }
        
      return next(c)
    }
  })

  /**
   * Handlers
   */
  bot.Handle("/start", func(c tele.Context) error {
    if c.Chat().Type != tele.ChatPrivate {
      return nil
    }
  
    messageText := ""
    buttonText := ""

    switch strings.ToLower((c.Sender().LanguageCode + "xx")[:2]) {
    case "uk":
      messageText= "👋 Привіт! Щоб продовжити, запусти застосунок за кнопкою нижче:"
      buttonText = "Запустити"
    case "en":
      messageText = "👋 Hi! To continue, launch the app using the button below:"
      buttonText = "Launch"
    default:
      messageText = "👋 Привет! Для продолжения запусти приложение по кнопке ниже:"
      buttonText = "Запустить"
    }

    btn := tele.InlineButton{
      Text: buttonText,
      WebApp: &tele.WebApp{
        URL: "https://" + appDomain,
      },
    }

    p.options.AuthPlugin.AuthBotUser(*c.Sender(), nil, nil)

  	return c.Send(messageText, &tele.SendOptions{
      ReplyMarkup: &tele.ReplyMarkup{
        InlineKeyboard: [][]tele.InlineButton{{btn}},
      },
    })
  })

  bot.Handle(tele.OnMyChatMember, func(c tele.Context) error {
    // No role change, do nothing
    if c.ChatMember().NewChatMember.Role == c.ChatMember().OldChatMember.Role {
      return nil
    }

    switch c.ChatMember().NewChatMember.Role {
    case tele.Administrator, tele.Creator:
      p.app.Logger().Info(
        "Telegram bot joined chat as administrator:",
        "ChatId", c.Chat().ID,
        "Title", c.Chat().Title,
      )

    case tele.Left:
    case tele.Kicked:
      p.app.Logger().Info(
        "Telegram bot left chat:",
        "ChatId", c.Chat().ID,
        "Title", c.Chat().Title,
      )

    case tele.Restricted:
      p.app.Logger().Info(
        "Telegram bot restricted in chat & left:",
        "ChatId", c.Chat().ID,
        "Title", c.Chat().Title,
      )
      return c.Bot().Leave(c.Chat())

    default:
      p.app.Logger().Info(
        "Telegram bot received unknown chat member update:",
        "ChatId", c.Chat().ID,
        "Title", c.Chat().Title,
      )
      return c.Bot().Leave(c.Chat())
    }

    return nil
  })

  bot.Handle(tele.OnChatJoinRequest, func(c tele.Context) error {
    joinRequest := c.ChatJoinRequest()
    p.app.Logger().Info(
      "Telegram bot received chat join request",
      "ChatID", joinRequest.Chat.ID,
      "ChatTitle", joinRequest.Chat.Title,
      "FirstName", joinRequest.Sender.FirstName,
      "LastName", joinRequest.Sender.LastName,
      "Username", joinRequest.Sender.Username,
      "LanguageCode", joinRequest.Sender.LanguageCode,
      "UserID", joinRequest.Sender.ID,
      "IsBot", joinRequest.Sender.IsBot,
    )

    user, err := p.options.AuthPlugin.AuthBotUser(*joinRequest.Sender, nil, nil)
    if err != nil {
      p.app.Logger().Warn("Failed to authenticate user from join request:", "Err", err, "UserID", joinRequest.Sender.ID)
      return nil
    }

    // Main channel
    if c.Chat().ID == int64(p.appConfig.GetInt("telegramChannelId")) {
      p.options.AuthPlugin.UpdateUserJoinPending(user, true)
    }

    return nil
  })

  bot.Handle(tele.OnChatMember, func(c tele.Context) error {
    member := c.ChatMember().NewChatMember

    switch c.Chat().ID {
    case int64(p.appConfig.GetInt("telegramChannelId")):
      p.options.AuthPlugin.AuthBotUser(*member.User, &member.Role, nil)
    case int64(p.appConfig.GetInt("telegramPremiumChannelId")):
      premium := false
      if member.Role == tele.Administrator || member.Role == tele.Creator || member.Role == tele.Member {
        premium = true
      }
      p.options.AuthPlugin.AuthBotUser(*member.User, nil, &premium)
    }

    return nil
  })

  /**
   * Cron jobs
   */
  p.app.Cron().MustAdd("telegram_bot_channel_users", p.options.CronExpr, func() {
    records, err := p.app.FindRecordsByFilter("users", "synced < {:synced}", "+synced", p.options.CronUsersPerRun, 0, dbx.Params{
      "synced": time.Now().Add(-p.options.CronUserSyncInterval).Format(time.RFC3339),
    })

    if err != nil {
      p.app.Logger().Warn("Failed to fetch users for Telegram bot channel update:", "Err", err)
      return
    }

    // Check if has access to the main channel
    mainChannel, err := bot.ChatByID(int64(p.appConfig.GetInt("telegramChannelId")))
    if err != nil {
      p.app.Logger().Warn("Telegram bot is not a member of the main channel, skipping user sync")
      return
    }

    // Check if has access to the premium channel
    premiumChannel, err := bot.ChatByID(int64(p.appConfig.GetInt("telegramPremiumChannelId")))
    if err != nil && premiumChannel != nil {
      p.app.Logger().Warn("Telegram bot is not a member of the premium channel, skipping premium user sync")
    }

    for _, record := range records {
      telegramId := int64(record.GetInt("telegramId"))

      // Main channel
      mainMember, err := bot.ChatMemberOf(mainChannel, &tele.User{ ID: telegramId })
      if err != nil && strings.Contains(err.Error(), "PARTICIPANT_ID_INVALID") {
        p.options.AuthPlugin.WhenUserLeft(record)
        continue
      } else if err != nil {
        p.app.Logger().Warn("Failed to fetch user from main channel:", "Err", err, "UserId", telegramId)
        continue
      }

      // Premium channel
      if premiumChannel != nil {
        premiumMember, err := bot.ChatMemberOf(premiumChannel, &tele.User{ ID: telegramId })

        if err != nil && strings.Contains(err.Error(), "PARTICIPANT_ID_INVALID") {
          record.Set("premium", false)
        } else if err != nil {
          p.app.Logger().Warn("Failed to fetch user from premium channel:", "Err", err, "UserId", telegramId)
        } else if premiumMember.Role == tele.Left || premiumMember.Role == tele.Kicked || premiumMember.Role == tele.Restricted {
          record.Set("premium", false)
        } else {
          record.Set("premium", true)
        }
      }

      // Save updated user record
      premium := record.GetBool("premium")
      p.options.AuthPlugin.AuthBotUser(*mainMember.User, &mainMember.Role, &premium);
    }
  })

  // Start the bot
  p.app.Logger().Info("Telegram bot authorized on account", "Username", bot.Me.Username)
  go bot.Start()
}