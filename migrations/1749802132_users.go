package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(func(app core.App) error {
		// Remove default "users" collection
		collection, err := app.FindCollectionByNameOrId("users")
		if err == nil {
			err := app.Delete(collection)
			if err != nil {
				return err
			}
		}

		// Create new "users" collection with custom fields
		collection = core.NewAuthCollection("users")

		// Rules
		collection.ListRule = types.Pointer("id = @request.auth.id || @request.auth.role = 'admin'")
		collection.ViewRule = types.Pointer("id = @request.auth.id || @request.auth.role = 'admin'")
		collection.ManageRule = types.Pointer("@request.auth.role = 'admin'")

		// Default fields (cant be removed)
		collection.Fields.GetByName("email").SetHidden(true)
		collection.Fields.GetByName("password").SetHidden(true)
		collection.Fields.GetByName("verified").SetHidden(true)
		collection.Fields.GetByName("emailVisibility").SetHidden(true)
		collection.Fields.GetByName("tokenKey").SetHidden(true)

		// Fields
		collection.Fields.Add(
			&core.NumberField{
				Name:     "telegramId",
				Required: true,
				OnlyInt:  true,
			},
			&core.TextField{
				Name:     "telegramUsername",
				Required: false,
				Max:      128,
			},
			&core.TextField{
				Name:     "name",
				Required: false,
				Max:      128,
			},
			&core.TextField{
				Name:     "language",
				Required: false,
				Max:      6,
			},
			&core.SelectField{
				Name:     "role",
				Required: true,
				Values:   []string{"guest", "user", "admin"},
			},
			&core.BoolField{
				Name:     "premium",
				Required: false,
			},
			&core.BoolField{
				Name:     "joinPending",
				Required: false,
			},
			&core.TextField{
				Name:     "avatarHash",
				Required: false,
				Hidden:   true,
				Max:      32,
			},
			&core.FileField{
				Name:     "avatar",
				Required: false,
				Thumbs:   []string{"250x250"},
				MimeTypes: []string{
					"image/jpeg",
					"image/png",
					"image/gif",
					"image/webp",
					"image/svg+xml",
					"image/avif",
					"image/apng",
					"image/avif",
				},
			},
			&core.AutodateField{
				Name:     "created",
				OnCreate: true,
			},
			&core.AutodateField{
				Name:     "updated",
				OnCreate: true,
				OnUpdate: true,
			},
			&core.DateField{
				Name:     "synced",
				Required: true,
			},
		)

		// Disable default auth
		collection.PasswordAuth.Enabled = false
		collection.AuthAlert.Enabled = false
		collection.AuthToken.Duration = 2419200 // 28 days

		// Indexes
		collection.AddIndex("idx_users__telegram_id", true, "telegramId", "")

		return app.Save(collection)
	}, func(app core.App) error {
		// Rollback migration
		collection, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}

		return app.Delete(collection)
	})
}
