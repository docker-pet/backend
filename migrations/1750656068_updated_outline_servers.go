package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("outline_servers")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"listRule": "@request.auth.role = 'user' || @request.auth.role = 'admin'",
			"viewRule": "@request.auth.role = 'user' || @request.auth.role = 'admin'"
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("outline_servers")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"listRule": "@request.auth.role = 'user'",
			"viewRule": "@request.auth.role = 'user'"
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
