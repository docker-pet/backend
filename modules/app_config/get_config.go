package app_config

import (
	"github.com/docker-pet/backend/models"
	"github.com/pocketbase/pocketbase/core"
)

func (m *AppConfigModule) AppConfig() *models.AppConfig {
	if m.currentConfig != nil {
		return m.currentConfig
	}

	record, err := m.Ctx.App.FindFirstRecordByFilter("app", "")
	if err != nil {
		panic("Failed to get app config record: " + err.Error())
	}

	m.currentConfig = ProxyAppConfig(record)
	return m.currentConfig
}


func ProxyAppConfig(record *core.Record) *models.AppConfig {
	appConfig := &models.AppConfig{}
	appConfig.SetProxyRecord(record)
	return appConfig
}
