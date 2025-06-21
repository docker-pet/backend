package telegram_bot

import (
	"fmt"
	"net/http"

	"github.com/pocketbase/pocketbase/core"
	tele "gopkg.in/telebot.v4"
)

func (m *TelegramBotModule) newBot(e *core.ServeEvent) (*tele.Bot, error) {
	endpointPath := fmt.Sprintf("/api/telegram_bot/%s", m.appConfig.AppConfig().TelegramBotToken())
	endpointUrl := fmt.Sprintf("https://%s%s", m.appConfig.AppConfig().AppDomain(), endpointPath)

	// Webhook
	webhook := &tele.Webhook{
		Endpoint: &tele.WebhookEndpoint{
			PublicURL: endpointUrl,
		},

		Listen: "",

		AllowedUpdates: []string{
			"my_chat_member",
			"chat_member",
			"chat_join_request",
			"message",
		},
	}

	// Bot
	bot, err := tele.NewBot(tele.Settings{
		Token:  m.appConfig.AppConfig().TelegramBotToken(),
		Poller: webhook,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to initialize Telegram bot: %w", err)
	}

	// Webhook Pooler
	_, err = bot.Webhook()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Telegram bot webhook: %w", err)
	}

	// HTTP Endpoint
	e.Router.POST(endpointPath, func(ctx *core.RequestEvent) error {
		webhook.ServeHTTP(ctx.Response, ctx.Request)
		return ctx.JSON(http.StatusOK, map[string]bool{"ok": true})
	})

	// Initialized
	return bot, nil
}
