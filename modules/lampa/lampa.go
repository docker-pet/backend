package lampa

import (
	"log/slog"

	"github.com/docker-pet/backend/core"
	"github.com/docker-pet/backend/models"
	"github.com/docker-pet/backend/modules/app_config"
	"github.com/docker-pet/backend/modules/users"
)

type Config struct {
	StoragePath string
}

type LampaModule struct {
	Ctx    *core.AppContext
	Config *Config
	Logger *slog.Logger

	users              *users.UsersModule
	appConfig          *app_config.AppConfigModule
	currentLampaConfig *models.LampaConfig
}

func (m *LampaModule) Name() string   { return "lampa" }
func (m *LampaModule) Deps() []string { return []string{"users", "app_config"} }
func (m *LampaModule) Init(ctx *core.AppContext, cfg any) error {
	m.Ctx = ctx
	m.Config = cfg.(*Config)
	m.Logger = ctx.App.Logger().WithGroup(m.Name())
	m.users = m.Ctx.Modules["users"].(*users.UsersModule)
	m.appConfig = m.Ctx.Modules["app_config"].(*app_config.AppConfigModule)

	m.watchConfigChanges()
	m.watchUsersChanges()
	go m.afterInit()

	m.Logger.Info("Lampa module initialized", "Config", m.Config)
	return nil
}
