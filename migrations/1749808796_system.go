package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
        settings := app.Settings()

		settings.Meta.AppName = "Docker Pet"
		settings.Meta.AppURL = "https://github.com/docker-pet"
		settings.Meta.HideControls = true

		settings.TrustedProxy.Headers = []string{"X-Forwarded-For"}

		settings.Logs.MaxDays = 14
		settings.Logs.LogIP = true
		settings.Logs.LogAuthId = true

		settings.Backups.Cron = "40 4 * * *" // every day at 04:40 UTC
		settings.Backups.CronMaxKeep = 14 // keep 7 backups

		return app.Save(settings)
	}, nil)
}
