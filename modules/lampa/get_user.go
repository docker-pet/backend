package lampa

import (
	"sort"

	"github.com/docker-pet/backend/models"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func (m *LampaModule) GetAllLampaUsers() ([]*models.LampaUser, error) {
	records, err := m.Ctx.App.FindAllRecords("lampa_users")
	if err != nil {
		return nil, err
	}

	// Proxy each record to LampaUser model
	lampaUsers := make([]*models.LampaUser, len(records))
	for i, record := range records {
		lampaUsers[i] = ProxyLampaUser(record)
	}

	// Sort
	sort.Slice(lampaUsers, func(i, j int) bool {
		return lampaUsers[i].Id < lampaUsers[j].Id
	})

	return lampaUsers, nil
}

func (m *LampaModule) GetLampaUserByUserId(userId string) (*models.LampaUser, error) {
	record, err := m.Ctx.App.FindFirstRecordByFilter(
		"lampa_users",
		"user={:userId}",
		dbx.Params{"userId": userId},
	)

	if err != nil {
		return nil, err
	}

	return ProxyLampaUser(record), nil
}

func ProxyLampaUser(record *core.Record) *models.LampaUser {
	lampaUser := &models.LampaUser{}
	lampaUser.SetProxyRecord(record)
	return lampaUser
}
