package migrations

import (
	"github.com/biter777/countries"
	"github.com/docker-pet/backend/helpers"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(func(app core.App) error {
		// Outline servers configuration migration
		collection := core.NewBaseCollection("outline_servers")

		// Rules
		collection.ListRule = types.Pointer("@request.auth.role = 'user'")
		collection.ViewRule = types.Pointer("@request.auth.role = 'user'")
		collection.ManageRule = types.Pointer("@request.auth.role = 'admin'")

		// Fields
		collection.Fields.Add(
			&core.TextField{
				Name:     "slug",
				Required: true,
				Min:      2,
				Max:      50,
				Pattern:  "^[a-z0-9-]+$",
			},
			&core.SelectField{
				Name:      "country",
				Required:  true,
				MaxSelect: 1,
				// TODO: Refresh country codes occasionally (ISO 3166 can change).
				Values: helpers.GetCountryCodes(),
			},
			&core.JSONField{
				Name:     "title",
				Required: false,
			},
			&core.JSONField{
				Name:     "description",
				Required: false,
			},
			&core.FileField{
				Name:      "banner",
				Required:  false,
				MimeTypes: []string{"image/jpeg", "image/png"},
				MaxSelect: 1,
			},
			&core.BoolField{
				Name: "enabled",
			},
			&core.BoolField{
				Name: "available",
			},
			&core.BoolField{
				Name: "premium",
			},
			&core.TextField{
				Name:     "overrideDomain",
				Required: false,
				Max:      256,
				Pattern:  "^([a-z0-9]+(-[a-z0-9]+)*\\.)+[a-z]{2,}$",
			},
			&core.TextField{
				Name:   "tcpPath",
				Max:    64,
				Hidden: true,
			},
			&core.NumberField{
				Name:   "tcpPort",
				OnlyInt: true,
				Hidden: true,
			},
			&core.TextField{
				Name:     "tcpPrefix",
				Required: false,
			},
			&core.TextField{
				Name:   "udpPath",
				Max:    64,
				Hidden: true,
			},
			&core.NumberField{
				Name:   "udpPort",
				OnlyInt: true,
				Hidden: true,
			},
			&core.TextField{
				Name:     "udpPrefix",
				Required: false,
			},
			&core.TextField{
				Name:   "servicePath",
				Max:    64,
				Hidden: true,
			},
			&core.TextField{
				Name:   "servicePassword",
				Max:    64,
				Hidden: true,
			},
			&core.TextField{
				Name:   "metricsSecret",
				Max:    64,
				Hidden: true,
			},
			&core.AutodateField{
				Name:     "created",
				OnCreate: true,
			},
		)

		// Indexes
		collection.AddIndex("idx_outline_servers__slug", true, "slug", "")
		collection.AddIndex("idx_outline_servers__enabled_premium", false, "`enabled`, `premium`", "")

		// Save
		err := app.Save(collection)
		if err != nil {
			return err
		}

		// Create demo server
		record := core.NewRecord(collection)
		record.Set("slug", "outline")
		record.Set("country", countries.Ukraine.Info().Alpha2)
		record.Set("title", `{"en": "Test Outline Server", "uk": "Тестовий Outline-сервер", "ru": "Тестовый Outline Сервер"}`)
		record.Set("description", `{"en": "Remove me", "uk": "Видали мене", "ru": "Удали меня"}`)
		record.Set("enabled", false)

		if err := app.Save(record); err != nil {
			return err
		}

		// Define default values for the demo server
		record.Set("servicePath", "jtWPhZocfhlIWQivW4rbrTatIi0ZtxzY")
		record.Set("servicePassword", "password")
		return app.Save(record)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("outline_servers")
		if err != nil {
			return err
		}

		return app.Delete(collection)
	})
}
