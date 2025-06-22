package migrations

import (
	"github.com/docker-pet/backend/modules/users"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Users collection
		collection, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}

		// Add outline_token field
		collection.Fields.Add(
			&core.TextField{
				Name:     "outlineToken",
				Required: false,
				Max:      64,
				Hidden:   false,
			},
		)

		if err := app.Save(collection); err != nil {
			return err
		}

		// Set default value for outline_token field
		records, err := app.FindAllRecords(collection)
		if err != nil {
			return err
		}
		for _, record := range records {
			user := users.ProxyUser(record)
			if user == nil {
				continue
			}

			user.GenerateOutlineToken()
			if err := app.Save(record); err != nil {
				return err
			}
		}

		return nil
	}, func(app core.App) error {
		// Users collection
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}

		// Remove outline_token field
		users.Fields.RemoveByName("outlineToken")

		return app.Save(users)
	})
}
