package lampa

import (
	"fmt"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/zmwangx/debounce"
)

func (m *LampaModule) watchConfigChanges() {
	buildInitConfigDebounced, _ := debounce.Debounce(
		func() { m.BuildInitConfig() },
		10*time.Second,
		debounce.WithLeading(true),
		debounce.WithTrailing(true),
	)

	// Delete
	m.Ctx.App.OnRecordDelete("lampa").BindFunc(func(e *core.RecordEvent) error {
		if e.Record.Id != m.LampaConfig().Id {
			return e.Next()
		}

		m.Logger.Warn("Attempt to delete the current lampa config record", "RecordId", e.Record.Id)
		return fmt.Errorf("cannot delete the current lampa config record: %s", e.Record.Id)
	})

	// Create
	m.Ctx.App.OnRecordCreate("lampa").BindFunc(func(e *core.RecordEvent) error {
		if m.currentLampaConfig == nil {
			return e.Next()
		}

		m.Logger.Warn("Attempt to create a new lampa config record while one already exists", "RecordId", m.LampaConfig().Id)
		return fmt.Errorf("cannot create a new lampa config record while one already exists: %s", m.LampaConfig().Id)
	})

	// Update
	m.Ctx.App.OnRecordAfterUpdateSuccess("lampa").BindFunc(func(e *core.RecordEvent) error {
		if e.Record.Id != m.LampaConfig().Id {
			return e.Next()
		}

		m.currentLampaConfig = ProxyLampaConfig(e.Record)
		m.Logger.Info("Lampa config updated successfully")

		m.BuildManifest()
		m.BuildPassword()
		buildInitConfigDebounced()

		return nil
	})

	// Lampa users collection events
	m.Ctx.App.OnRecordAfterCreateSuccess("lampa_users").BindFunc(func(e *core.RecordEvent) error {
		user := ProxyLampaUser(e.Record)
		m.Logger.Info("Lampa user created", "UserId", user.UserId())
		buildInitConfigDebounced()
		return e.Next()
	})

	m.Ctx.App.OnRecordAfterDeleteSuccess("lampa_users").BindFunc(func(e *core.RecordEvent) error {
		user := ProxyLampaUser(e.Record)
		m.Logger.Info("Lampa user deleted", "UserId", user.UserId())
		buildInitConfigDebounced()
		return e.Next()
	})

	m.Ctx.App.OnRecordAfterUpdateSuccess("lampa_users").BindFunc(func(e *core.RecordEvent) error {
		user := ProxyLampaUser(e.Record)
		m.Logger.Info("Lampa user updated", "UserId", user.UserId())
		buildInitConfigDebounced()
		return e.Next()
	})

}
