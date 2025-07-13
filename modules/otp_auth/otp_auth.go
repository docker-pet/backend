package otp_auth

import (
	"log/slog"
	"time"

	"github.com/docker-pet/backend/core"
	"github.com/docker-pet/backend/modules/app_config"
	"github.com/docker-pet/backend/modules/lampa"
	"github.com/docker-pet/backend/modules/users"
)

type Config struct {
	SessionVerifyInterval             time.Duration // Interval for verifying the authorized OTP session
	AuthSessionLifetime               time.Duration // Duration after which the OTP session expires
	ExpiredAuthSessionCleanupInterval time.Duration // Interval at which expired OTP sessions are cleaned up
	MaxPinGenerationAttempts          int           // Maximum attempts to generate a unique PIN code

}

type OtpAuthModule struct {
	Ctx    *core.AppContext
	Config *Config
	Logger *slog.Logger

	appConfig *app_config.AppConfigModule
	users     *users.UsersModule
	lampa     *lampa.LampaModule
	keychain  *KeyChain
}

func (m *OtpAuthModule) Name() string                  { return "otp_auth" }
func (m *OtpAuthModule) Deps() []string                { return []string{"users", "app_config"} }
func (m *OtpAuthModule) SetLogger(logger *slog.Logger) { m.Logger = logger }
func (m *OtpAuthModule) Init(ctx *core.AppContext, logger *slog.Logger, cfg any) error {
	m.Ctx = ctx
	m.Config = cfg.(*Config)
	m.Logger = logger
	m.appConfig = m.Ctx.Modules["app_config"].(*app_config.AppConfigModule)
	m.users = m.Ctx.Modules["users"].(*users.UsersModule)
	m.lampa = m.Ctx.Modules["lampa"].(*lampa.LampaModule)
	m.keychain = NewKeyChain(&KeyChainOptions{
		Expiration:      m.Config.AuthSessionLifetime,
		CleanupInterval: m.Config.ExpiredAuthSessionCleanupInterval,
	})

	m.registerOtpConfirmEndpoint()
	m.registerOtpVerifyEndpoint()
	m.registerOtpUserEndpoint()
	m.registerOtpSessionEndpoint()

	m.Logger.Info("OTP Auth module initialized", "Config", m.Config)
	return nil
}
