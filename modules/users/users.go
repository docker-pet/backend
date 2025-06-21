package users

import (
	"log/slog"
	"github.com/docker-pet/backend/core"
)

type Config struct{}

type UsersModule struct {
	Ctx    *core.AppContext
	Config *Config
	Logger *slog.Logger
}

func (m *UsersModule) Name() string   { return "users" }
func (m *UsersModule) Deps() []string { return nil }
func (m *UsersModule) Init(ctx *core.AppContext, cfg any) error {
	m.Ctx = ctx
	m.Config = cfg.(*Config)
	m.Logger = ctx.App.Logger().WithGroup(m.Name())


	m.Logger.Info("Users module initialized", "Config", m.Config)
	return nil
}
