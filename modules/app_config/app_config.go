package app_config

import (
	"log/slog"

	"github.com/docker-pet/backend/core"
	"github.com/docker-pet/backend/models"
)

type Config struct{}

type AppConfigModule struct {
	Ctx    *core.AppContext
	Config *Config
	Logger *slog.Logger

	currentConfig *models.AppConfig
}

func (m *AppConfigModule) Name() string   { return "app_config" }
func (m *AppConfigModule) Deps() []string { return nil }
func (m *AppConfigModule) Init(ctx *core.AppContext, cfg any) error {
	m.Ctx = ctx
	m.Config = cfg.(*Config)
	m.Logger = ctx.App.Logger().WithGroup(m.Name())

	m.watchChanges()

	m.Logger.Info("App config module initialized", "Config", m.Config)
	return nil
}
