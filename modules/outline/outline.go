package outline

import (
	"log/slog"
	"time"

	"github.com/docker-pet/backend/core"
	"github.com/docker-pet/backend/modules/app_config"
	"github.com/docker-pet/backend/modules/users"
	"github.com/pocketbase/pocketbase/tools/security"
)

type Config struct {
	OutlineStoragePath        string
	OutlineCipher             string
	OutlineTechnicalKeyName   string
	OutlineTechnicalKeySecret string

	PrometheusStoragePath       string
	PrometheusJobName           string
	PrometheusJobManagedByLabel string

	CaddyCloudflareApiToken string // Is the API token for Cloudflare. If not set, will use a placeholder.

	MetricsProxySecret string // Is the secret for the metrics proxy endpoint. If not set, a new one will be generated (recommended).

	TokenStoreSlidingTTL      time.Duration
	TokenStoreAbsoluteTTL     time.Duration
	TokenStoreCleanupInterval time.Duration
}

type OutlineModule struct {
	Ctx    *core.AppContext
	Config *Config
	Logger *slog.Logger

	users     *users.UsersModule
	appConfig *app_config.AppConfigModule

	tokenStore *TokenStore
}

func (m *OutlineModule) Name() string                  { return "outline" }
func (m *OutlineModule) Deps() []string                { return []string{"users", "app_config"} }
func (m *OutlineModule) SetLogger(logger *slog.Logger) { m.Logger = logger }
func (m *OutlineModule) Init(ctx *core.AppContext, logger *slog.Logger, cfg any) error {
	m.Ctx = ctx
	m.Config = cfg.(*Config)
	m.Logger = logger
	m.users = m.Ctx.Modules["users"].(*users.UsersModule)
	m.appConfig = m.Ctx.Modules["app_config"].(*app_config.AppConfigModule)
	m.tokenStore = NewTokenStore(
		m.Config.TokenStoreSlidingTTL,
		m.Config.TokenStoreAbsoluteTTL,
		m.Config.TokenStoreCleanupInterval,
		m.Config.OutlineTechnicalKeyName,
		m.Config.OutlineTechnicalKeySecret,
	)

	// Generate metrics proxy secret if not set
	if m.Config.MetricsProxySecret == "" {
		m.Config.MetricsProxySecret = security.RandomString(32)
	}

	m.registerMetrixProxyEndpoint()
	m.registerOutlineConnectEndpoint()
	m.registerSettingsEndpoint()

	m.watchConfigChanges()
	m.watchKeysChanges()

	m.Logger.Info("Outline module initialized", "Config", m.Config)
	return nil
}
