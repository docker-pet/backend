package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/security"
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
				Name:     "dlnaEnabled",
			},
			&core.BoolField{
				Name:     "sisiEnabled",
			},
			&core.BoolField{
				Name:     "tmdbProxyEnabled",
			},
			&core.BoolField{
				Name:     "torrServerEnabled",
			},
			&core.BoolField{
				Name:     "serverProxyEnabled",
			},
			&core.BoolField{
				Name:     "cubEnabled",
			},
			&core.BoolField{
				Name:     "tracksEnabled",
			},
			&core.BoolField{
				Name:     "onlineEnabled",
			},
            &core.TextField{
                Name:     "adminPassword",
                Required: true,
				Hidden:   true,
            },
            &core.JSONField{
                Name:     "configInit",
                Required: false,
				Hidden: true,
            },
		)

		// Save
		if err := app.Save(collection); err != nil {
			return err
		}

        // Add first record
		record := core.NewRecord(collection)

		// Flags
		record.Set("dlnaEnabled", false)
		record.Set("sisiEnabled", true)
		record.Set("tmdbProxyEnabled", true)
		record.Set("torrServerEnabled", false)
		record.Set("serverProxyEnabled", true)
		record.Set("cubEnabled", true)
		record.Set("tracksEnabled", false)
		record.Set("onlineEnabled", true)

		// Init config
        record.Set("configInit", `{}`)
        
		// Password
		record.Set("adminPassword", security.RandomString(30))

        return app.Save(record)
	}, func(app core.App) error {
        collection, err := app.FindCollectionByNameOrId("lampa")
        if err != nil {
            return err
        }

        return app.Delete(collection)
	})
}
