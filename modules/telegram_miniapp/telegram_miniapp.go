package telegram_miniapp

import (
	"log/slog"
	"time"

	"github.com/docker-pet/backend/core"
	"github.com/docker-pet/backend/modules/app_config"
	"github.com/docker-pet/backend/modules/users"
)

type Config struct {
	AuthTokenLifetime time.Duration
}

type TelegramMiniappModule struct {
	Ctx    *core.AppContext
	Config *Config
	Logger *slog.Logger

	users     *users.UsersModule
	appConfig *app_config.AppConfigModule
}

func (m *TelegramMiniappModule) Name() string   { return "telegram_miniapp" }
func (m *TelegramMiniappModule) Deps() []string { return []string{"users", "app_config"} }
func (m *TelegramMiniappModule) Init(ctx *core.AppContext, cfg any) error {
	m.Ctx = ctx
	m.Config = cfg.(*Config)
	m.Logger = ctx.App.Logger().WithGroup(m.Name())
	m.users = m.Ctx.Modules["users"].(*users.UsersModule)
	m.appConfig = m.Ctx.Modules["app_config"].(*app_config.AppConfigModule)

	m.registerAuthVerifyEndpoint()

	m.Logger.Info("Telegram MiniApp module initialized", "Config", m.Config)
	return nil
}
