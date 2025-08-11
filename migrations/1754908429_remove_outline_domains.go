package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Outline servers collection
		collection, err := app.FindCollectionByNameOrId("outline_servers")
		if err != nil {
			return err
		}

		collection.Fields.RemoveByName("reverseDomain")
		collection.Fields.RemoveByName("overrideDomain")
		if err := app.Save(collection); err != nil {
			return err
		}

		return nil
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("outline_servers")
		if err != nil {
			return err
		}

		collection.Fields.Add(
			&core.TextField{
				Name:     "reverseDomain",
				Required: false,
				Hidden:   true,
			},
			&core.TextField{
				Name:     "overrideDomain",
				Required: false,
				Hidden:   true,
			},
		)

		if err := app.Save(collection); err != nil {
			return err
		}

		return nil
	})
}
