package main

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"

	_ "github.com/docker-pet/backend/migrations"
	restartAppPlugin "github.com/docker-pet/backend/plugins/restart_app"
	otpAuthPlugin "github.com/docker-pet/backend/plugins/otp_auth"
	telegramAuthPlugin "github.com/docker-pet/backend/plugins/telegram_auth"
	telegramBotPlugin "github.com/docker-pet/backend/plugins/telegram_bot"
)

func main() {
    app := pocketbase.New()
    isGoRun := strings.HasPrefix(os.Args[0], os.TempDir())

	// Migrations
    migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
        Automigrate: isGoRun,
    })

	// App restart on configuration update
	restartAppPlugin.Register(app, &restartAppPlugin.Options{})

	// Setup telegram mini apps auth
	authPlugin := telegramAuthPlugin.Register(app, &telegramAuthPlugin.Options{
		CollectionKey: "users",
	})

	// OTP Auth
	otpAuthPlugin.Register(app, &otpAuthPlugin.Options{
		AuthVerifyInterval:       time.Minute * 7,
		Expiration:               time.Minute * 5,
		CleanupInterval:          time.Minute * 10,
		MaxPinGenerationAttempts: 10,
	})

	// Setup telegram bot
	telegramBotPlugin.Register(app, &telegramBotPlugin.Options{
		AuthPlugin: authPlugin,
		CronExpr:   "*/15 * * * *",
		CronUsersPerRun: 10,
		CronUserSyncInterval: time.Minute * 60,
	})

    if err := app.Start(); err != nil {
        log.Fatal(err)
    }
}
