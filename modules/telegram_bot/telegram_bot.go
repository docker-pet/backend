package telegram_bot

import (
	"log/slog"
	"time"

	"github.com/docker-pet/backend/core"
	"github.com/docker-pet/backend/modules/app_config"
	"github.com/docker-pet/backend/modules/users"
	pbCore "github.com/pocketbase/pocketbase/core"
	tele "gopkg.in/telebot.v4"
)

type Config struct {
	CronUserSyncInterval   time.Duration
	CronUserSyncExpression string
	CronUsersPerSync       int
}

type TelegramBotModule struct {
	Ctx    *core.AppContext
	Config *Config
	Logger *slog.Logger

	appConfig *app_config.AppConfigModule
	users     *users.UsersModule

	Bot *tele.Bot
}

func (m *TelegramBotModule) Name() string                  { return "telegram_bot" }
func (m *TelegramBotModule) Deps() []string                { return []string{"users", "app_config"} }
func (m *TelegramBotModule) SetLogger(logger *slog.Logger) { m.Logger = logger }
func (m *TelegramBotModule) Init(ctx *core.AppContext, logger *slog.Logger, cfg any) error {
	m.Ctx = ctx
	m.Config = cfg.(*Config)
	m.Logger = logger
	m.appConfig = m.Ctx.Modules["app_config"].(*app_config.AppConfigModule)
	m.users = m.Ctx.Modules["users"].(*users.UsersModule)

	m.useUsersRevalidateCron()

	m.Ctx.App.OnServe().BindFunc(func(e *pbCore.ServeEvent) error {
		// Initialize bot
		bot, err := m.newBot(e)
		if err != nil {
			m.Logger.Warn(
				"Failed to initialize Telegram bot",
				"Error", err,
				"Config", m.Config,
			)
			return e.Next()
		}
		m.Bot = bot

		// Handlers
		m.useAccessMiddleware()
		m.useOnChatMember()
		m.useOnChatJoinRequest()
		m.useOnMyChatMember()
		m.useStartCommand()
		m.appConfig.SetBotUsername(m.Bot.Me.Username)

		// Start
		go m.Bot.Start()
		m.Logger.Info(
			"Telegram Bot module initialized",
			"Config", m.Config,
			"BotUsername", m.Bot.Me.Username,
		)

		return e.Next()
	})

	return nil
}
