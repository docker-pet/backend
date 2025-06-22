package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
	"github.com/pocketbase/pocketbase/tools/security"
)

func init() {
	m.Register(func(app core.App) error {
		// Application configuration migration
		collection := core.NewBaseCollection("app")

		// Rules
		collection.ListRule = types.Pointer("")
        collection.ViewRule = types.Pointer("")

		// Fields
		collection.Fields.Add(
			&core.TextField{
				Name:     "appDomain",
				Required: true,
				Max: 256,
				Pattern: "^([a-z0-9]+(-[a-z0-9]+)*\\.)+[a-z]{2,}$",
			},
            &core.TextField{
                Name:     "telegramBotToken",
                Required: true,
				Max: 256,
				Hidden: true,
            },
            &core.NumberField{
                Name:     "telegramChannelId",
                Required: true,
				OnlyInt: true,
				Hidden: true,
            },
			&core.URLField{
				Name:    "telegramChannelInviteLink",
				Required: true,
				OnlyDomains: []string{"t.me"},
			},
            &core.NumberField{
                Name:     "telegramPremiumChannelId",
                Required: true,
				OnlyInt: true,
				Hidden: true,
            },
			&core.URLField{
				Name:    "telegramPremiumChannelInviteLink",
				Required: true,
				OnlyDomains: []string{"t.me"},
			},
			&core.TextField{
				Name: "authSecret",
				Required: true,
				Hidden: true,
				Max: 64,
				Min: 32,
			},
			&core.TextField{
				Name:     "authCookieName",
				Required: true,
				Hidden: true,
				Min: 6,
				Max: 32,
			},
            &core.NumberField{
                Name:     "authPinLength",
                Required: true,
				OnlyInt: true,
            },
		)

		// Save
		if err := app.Save(collection); err != nil {
			return err
		}

        // Add first record
		record := core.NewRecord(collection)

		// Set default values
        record.Set("appDomain", "dev.docker.pet")
		record.Set("telegramBotToken", "FILL_ME")
        record.Set("telegramChannelId", -1)
		record.Set("telegramChannelInviteLink", "https://t.me/+xxxxxxxXxxXXXxxx")
		record.Set("telegramPremiumChannelId", -1)
		record.Set("telegramPremiumChannelInviteLink", "https://t.me/+xxxxxxxXxxXXXxxx")
        record.Set("authSecret", security.RandomString(32))
		record.Set("authCookieName", "auth_" + security.RandomString(12))
		record.Set("authPinLength", 5)

        return app.Save(record)
	}, func(app core.App) error {
        collection, err := app.FindCollectionByNameOrId("app")
        if err != nil {
            return err
        }

        return app.Delete(collection)
	})
}
