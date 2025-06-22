package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(func(app core.App) error {
		// Users collection
		usersCollection, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}

		// Lampa users migration
		collection := core.NewBaseCollection("lampa_users")

		// Rules
		collection.ListRule = types.Pointer("@request.auth.role = 'admin'")
        collection.ViewRule = types.Pointer("@request.auth.role = 'admin'")
		collection.ManageRule = types.Pointer("@request.auth.role = 'admin'")

		// Fields
		collection.Fields.Add(
			&core.RelationField{
				Name:     "user",
				CollectionId: usersCollection.Id,
				Required: true,
				CascadeDelete: true,
				MinSelect: 1,
				MaxSelect: 1,
			},
			&core.TextField{
				Name:     "authKey",
				Pattern: "^[A-Za-z0-9]{32}$",
				Required: true,
				Hidden: true,
			},
			&core.BoolField{
				Name:    "disabled",
			},
		)

		// Indexes
		collection.AddIndex("idx_lampa_users__auth_key", true, "authKey", "")
		collection.AddIndex("idx_lampa_users__user_key", true, "user", "")

		// Save
		if err := app.Save(collection); err != nil {
			return err
		}

		return nil
	}, func(app core.App) error {
        collection, err := app.FindCollectionByNameOrId("lampa_users")
        if err != nil {
            return err
        }

        return app.Delete(collection)
	})
}
