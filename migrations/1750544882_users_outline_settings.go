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

		// Outline servers collection
		outlineServersCollection, err := app.FindCollectionByNameOrId("outline_servers")
		if err != nil {
			return err
		}

		// Fields
		collection.Fields.Add(
			&core.RelationField{
				Name:          "outlineServer",
				CollectionId:  outlineServersCollection.Id,
				Required:      false,
				CascadeDelete: false,
				MinSelect:     1,
				MaxSelect:     1,
			},
			&core.BoolField{
				Name:          "outlinePrefixEnabled",
				Required:      false,
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

		// Remove outline fields
		users.Fields.RemoveByName("outlineServer")
		users.Fields.RemoveByName("outlinePrefixEnabled")

		return app.Save(users)
	})
}
