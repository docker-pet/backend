package lampa

import (
	"github.com/docker-pet/backend/models"
	"github.com/pocketbase/pocketbase/core"
)

func (m *LampaModule) LampaConfig() *models.LampaConfig {
	if m.currentLampaConfig != nil {
		return m.currentLampaConfig
	}

	record, err := m.Ctx.App.FindFirstRecordByFilter("lampa", "")
	if err != nil {
		panic("Failed to get lampa config record: " + err.Error())
	}

	m.currentLampaConfig = ProxyLampaConfig(record)
	return m.currentLampaConfig
}

func ProxyLampaConfig(record *core.Record) *models.LampaConfig {
	lampaConfig := &models.LampaConfig{}
	lampaConfig.SetProxyRecord(record)
	return lampaConfig
}
