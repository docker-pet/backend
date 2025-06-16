package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(func(app core.App) error {
		// Lampa configuration migration
		collection := core.NewBaseCollection("lampa")

		// Rules
		collection.ListRule = types.Pointer("@request.auth.role = 'admin'")
        collection.ViewRule = types.Pointer("@request.auth.role = 'admin'")
		collection.ManageRule = types.Pointer("@request.auth.role = 'admin'")

		// Fields
		collection.Fields.Add(
			&core.BoolField{
				Name:    "enabled",
				Required: false,
			},
            &core.URLField{
                Name:     "link",
                Required: true,
            },
		)

		// Save
		if err := app.Save(collection); err != nil {
			return err
		}

        // Add first record
		record := core.NewRecord(collection)

		// Set default values
        record.Set("enabled", true)
        record.Set("link", "https://github.com/immisterio/Lampac")

        return app.Save(record)
	}, func(app core.App) error {
        collection, err := app.FindCollectionByNameOrId("lampa")
        if err != nil {
            return err
        }

        return app.Delete(collection)
	})
}
