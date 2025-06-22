package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	email := "admin@telegram.internal"
	password := "password"

	m.Register(func(app core.App) error {
        superusers, err := app.FindCollectionByNameOrId(core.CollectionNameSuperusers)
        if err != nil {
            return err
        }

        record := core.NewRecord(superusers)

        record.Set("email", email)
        record.Set("password", password)

        return app.Save(record)
	}, func(app core.App) error {
        record, _ := app.FindAuthRecordByEmail(core.CollectionNameSuperusers, email)
        if record == nil {
            return nil
        }

        return app.Delete(record)
	})
}
