package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Application collection
		collection, err := app.FindCollectionByNameOrId("app")
		if err != nil {
			return err
		}

		// Add field
		collection.Fields.Add(
			&core.TextField{
				Name:     "appDomainReverse",
				Required: false,
			},
		)

		if err := app.Save(collection); err != nil {
			return err
		}

		// App config
		config, err := app.FindFirstRecordByFilter("app", "")
		if err != nil {
			return err
		}

		config.Set("appTitle", "Docker Pet")
		return app.Save(config)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("app")
		if err != nil {
			return err
		}

		collection.Fields.RemoveByName("appDomainReverse")

		return app.Save(collection)
	})
}

//
