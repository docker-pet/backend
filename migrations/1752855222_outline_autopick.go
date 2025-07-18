package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Collection
		collection, err := app.FindCollectionByNameOrId("outline_servers")
		if err != nil {
			return err
		}

		// Fields
		collection.Fields.Add(
			&core.BoolField{
				Name:     "autopick",
				Required: false,
				Hidden:   true,
			},
		)

		return app.Save(collection)
	}, func(app core.App) error {
		// Collection
		collection, err := app.FindCollectionByNameOrId("outline_servers")
		if err != nil {
			return err
		}

		// Fields
		collection.Fields.RemoveByName("autopick")

		return app.Save(collection)
	})
}
