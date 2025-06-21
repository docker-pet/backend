package users

import (
	"github.com/docker-pet/backend/models"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func (m *UsersModule) GetUserByTelegramId(telegramId int64) (*models.User, error) {
	record, err := m.Ctx.App.FindFirstRecordByFilter(
		"users",
		"telegramId={:telegramId}",
		dbx.Params{"telegramId": telegramId},
	)

	if err != nil {
		return nil, err
	}

	return ProxyUser(record), nil
}

func (m *UsersModule) GetUserById(id string) (*models.User, error) {
	record, err := m.Ctx.App.FindRecordById("users", id)

	if err != nil {
		return nil, err
	}

	return ProxyUser(record), nil
}

func (m *UsersModule) GetAllUsers(exprs ...dbx.Expression) ([]*models.User, error) {
	records, err := m.Ctx.App.FindAllRecords("users", exprs...)
	if err != nil {
		return nil, err
	}

	lampaUsers := make([]*models.User, len(records))
	for i, record := range records {
		lampaUsers[i] = ProxyUser(record)
	}

	return lampaUsers, nil
}

func (m *UsersModule) FindUsersByFilter(filter string, sort string, limit int, offset int, params ...dbx.Params) ([]*models.User, error) {
	records, err := m.Ctx.App.FindRecordsByFilter("users", filter, sort, limit, offset, params...)
	if err != nil {
		return nil, err
	}

	lampaUsers := make([]*models.User, len(records))
	for i, record := range records {
		lampaUsers[i] = ProxyUser(record)
	}

	return lampaUsers, nil
}

func ProxyUser(record *core.Record) *models.User {
	user := &models.User{}
	user.SetProxyRecord(record)
	return user
}
