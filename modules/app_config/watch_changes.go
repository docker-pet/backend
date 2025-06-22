package app_config

import (
	"fmt"
	"time"

	"github.com/docker-pet/backend/models"
	"github.com/pocketbase/pocketbase/core"
)

func (m *AppConfigModule) watchChanges() {
	// Delete
	m.Ctx.App.OnRecordDelete("app").BindFunc(func(e *core.RecordEvent) error {
		if e.Record.Id != m.AppConfig().Id {
			return e.Next()
		}

		m.Logger.Warn("Attempt to delete the current app config record", "RecordId", e.Record.Id)
		return fmt.Errorf("cannot delete the current app config record: %s", e.Record.Id)
	})

	// Create
	m.Ctx.App.OnRecordCreate("app").BindFunc(func(e *core.RecordEvent) error {
		if m.currentConfig == nil {
			return e.Next()
		}

		m.Logger.Warn("Attempt to create a new app config record while one already exists", "RecordId", m.AppConfig().Id)
		return fmt.Errorf("cannot create a new app config record while one already exists: %s", m.AppConfig().Id)
	})

	// Update
	m.Ctx.App.OnRecordAfterUpdateSuccess("app").BindFunc(func(e *core.RecordEvent) error {
		if e.Record.Id != m.AppConfig().Id {
			return e.Next()
		}

		newConfig := ProxyAppConfig(e.Record)
		m.RestartAppIfNeeded(newConfig)
		m.currentConfig = newConfig

		m.Logger.Info("App config updated successfully")
		return nil
	})
}

func (m *AppConfigModule) RestartAppIfNeeded(config *models.AppConfig) {
	if m.AppConfig().TelegramBotToken() != config.TelegramBotToken() ||
		m.AppConfig().AuthPinLength() != config.AuthPinLength() ||
		m.AppConfig().AppDomain() != config.AppDomain() {
		m.Logger.Info("Restarting app due to config changes")
		go func() {
			time.Sleep(3 * time.Second)
			err := m.Ctx.App.Restart()
			if err != nil {
				panic(fmt.Errorf("failed to restart app: %w", err))
			}
		}()
	}
}
