package users

import (
	"github.com/docker-pet/backend/models"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

func (m *UsersModule) NewUser(telegramId int64) (*models.User, error) {
	collection, err := m.Ctx.App.FindCollectionByNameOrId("users")
	if err != nil {
		return nil, err
	}

	record := core.NewRecord(collection)
	user := ProxyUser(record)

	user.SetTelegramId(telegramId)
	user.SetRole(models.RoleGuest)
	user.SetSynced(types.NowDateTime().AddDate(-20, 0, 0))
	user.SetOutlinePrefixEnabled(false)
	user.SetOutlineReverseServerEnabled(true)

	return user, nil
}
