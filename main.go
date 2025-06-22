package main

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"github.com/pocketbase/pocketbase/tools/security"

	"github.com/docker-pet/backend/core"
	_ "github.com/docker-pet/backend/migrations"
	"github.com/docker-pet/backend/modules/app_config"
	"github.com/docker-pet/backend/modules/lampa"
	"github.com/docker-pet/backend/modules/otp_auth"
	"github.com/docker-pet/backend/modules/outline"
	"github.com/docker-pet/backend/modules/telegram_bot"
	"github.com/docker-pet/backend/modules/telegram_miniapp"
	"github.com/docker-pet/backend/modules/users"
)

func main() {
	app := pocketbase.New()
	isGoRun := strings.HasPrefix(os.Args[0], os.TempDir())

	// Migrations
	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		Automigrate: isGoRun,
	})

	// HTTP Client
	httpClient := core.NewHttpClient()
	defer httpClient.Close()

	// Modules
	ctx := &core.AppContext{
		App:        app,
		HttpClient: httpClient,
	}

	core.RegisterModule(&app_config.AppConfigModule{}, &app_config.Config{})

	core.RegisterModule(&users.UsersModule{}, &users.Config{})

	core.RegisterModule(&otp_auth.OtpAuthModule{}, &otp_auth.Config{
		SessionVerifyInterval:             time.Minute * 7,
		AuthSessionLifetime:               time.Minute * 5,
		ExpiredAuthSessionCleanupInterval: time.Minute * 15,
		MaxPinGenerationAttempts:          10,
	})

	core.RegisterModule(&lampa.LampaModule{}, &lampa.Config{
		StoragePath: "./generated/lampa",
	})

	core.RegisterModule(&telegram_bot.TelegramBotModule{}, &telegram_bot.Config{
		CronUserSyncExpression: "*/15 * * * *",
		CronUserSyncInterval:   time.Minute * 60,
		CronUsersPerSync:       10,
	})

	core.RegisterModule(&telegram_miniapp.TelegramMiniappModule{}, &telegram_miniapp.Config{
		AuthTokenLifetime: time.Hour * 12,
	})

	core.RegisterModule(&outline.OutlineModule{}, &outline.Config{
		OutlineStoragePath:        "./generated/outline",
		OutlineCipher:             "chacha20-ietf-poly1305",
		OutlineTechnicalKeyName:   "service",
		OutlineTechnicalKeySecret: security.RandomString(32),

		PrometheusStoragePath:       "./generated/prometheus",
		PrometheusJobName:           "outline",
		PrometheusJobManagedByLabel: "github.com/docker-pet",

		CaddyCloudflareApiToken:    os.Getenv("CLOUDFLARE_API_TOKEN"),

		TokenStoreSlidingTTL:      time.Hour * 4,
		TokenStoreAbsoluteTTL:     time.Hour * 48,
		TokenStoreCleanupInterval: time.Hour * 8,
	})

	// Star app
	err := core.InitModules(ctx)
	if err != nil {
		log.Fatal(err)
	} else if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
