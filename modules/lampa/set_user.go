package lampa

import (
	"github.com/docker-pet/backend/models"
	"github.com/pocketbase/pocketbase/core"
)

func (m *LampaModule) NewLampaUser(user *models.User) (*models.LampaUser, error) {
	collection, err := m.Ctx.App.FindCollectionByNameOrId("lampa_users")
	if err != nil {
		return nil, err
	}

	record := core.NewRecord(collection)
	lampaUser := ProxyLampaUser(record)

	lampaUser.SetUserId(user.Id)
	lampaUser.GenerateAuthKey()
	lampaUser.SetDisabled(user.Role() == models.RoleGuest)

	return lampaUser, nil
}
