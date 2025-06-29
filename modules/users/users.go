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

func (m *UsersModule) Name() string                  { return "users" }
func (m *UsersModule) Deps() []string                { return nil }
func (m *UsersModule) SetLogger(logger *slog.Logger) { m.Logger = logger }
func (m *UsersModule) Init(ctx *core.AppContext, logger *slog.Logger, cfg any) error {
	m.Ctx = ctx
	m.Config = cfg.(*Config)
	m.Logger = logger

	m.Logger.Info("Users module initialized", "Config", m.Config)
	return nil
}
