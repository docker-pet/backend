package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Users collection
		usersCollection, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}

		usersCollection.Fields.Add(
			&core.BoolField{
				Name:     "outlineReverseServerEnabled",
				Required: false,
			},
		)

		if err := app.Save(usersCollection); err != nil {
			return err
		}

		// Outline servers collection
		outlineCollection, err := app.FindCollectionByNameOrId("outline_servers")
		if err != nil {
			return err
		}

		outlineCollection.Fields.Add(
			&core.TextField{
				Name:     "reverseDomain",
				Required: false,
				Hidden: true,
			},
		)

		if err := app.Save(outlineCollection); err != nil {
			return err
		}

		return nil
	}, func(app core.App) error {
		// Users collection
		usersCollection, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}

		usersCollection.Fields.RemoveByName("outlineReverseServerEnabled")
		if err := app.Save(usersCollection); err != nil {
			return err
		}

		// Outline servers collection		
		outlineCollection, err := app.FindCollectionByNameOrId("outline_servers")
		if err != nil {
			return err
		}

		outlineCollection.Fields.RemoveByName("reverseDomain")
		if err := app.Save(outlineCollection); err != nil {
			return err
		}

		return nil
	})
}
