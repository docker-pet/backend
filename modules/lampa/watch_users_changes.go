package lampa

import (
	"github.com/docker-pet/backend/models"
	"github.com/docker-pet/backend/modules/users"
	"github.com/pocketbase/pocketbase/core"
)

func (m *LampaModule) watchUsersChanges() {
	// Check all users on app start
    m.Ctx.App.OnServe().BindFunc(func(e *core.ServeEvent) error {
		users, err := m.users.GetAllUsers()
		if err != nil {
			m.Logger.Error("Failed to get all users", "Error", err)
			return e.Next()
		}

		for _, user := range users {
			lampaUser, err := m.GetLampaUserByUserId(user.Id)
			needToSave := false

			// New user
			if err != nil {
				if user.Role() == models.RoleGuest {
					continue
				}
				
				lampaUser, err = m.NewLampaUser(user)
				needToSave = true
				if err != nil {
					m.Logger.Error(
						"Failed to create Lampa user for existing user",
						"UserId", user.Id,
						"Error", err,
					)
					continue
				}
			}

			// Has changed role
			disabled := user.Role() == models.RoleGuest
			if lampaUser.Disabled() != disabled {
				lampaUser.SetDisabled(disabled)
				needToSave = true
			}

			// Has changes
			if needToSave {
				if err := m.Ctx.App.Save(lampaUser); err != nil {
					m.Logger.Error(
						"Failed to save Lampa user",
						"UserId", user.Id,
						"LampaUserId", lampaUser.Id,
						"Error", err,
					)
				}
			}
		}

        return e.Next()
	})

	// User created
	m.Ctx.App.OnRecordAfterCreateSuccess("users").BindFunc(func(e *core.RecordEvent) error {
		user := users.ProxyUser(e.Record)
		if user.Role() == models.RoleGuest {
			return e.Next()
		}

		lampaUser, err := m.NewLampaUser(user)
		if err != nil {
			m.Logger.Error(
				"Failed to create Lampa user for new user",
				"UserId", user.Id,
				"Error", err,
			)
		}

		if err := m.Ctx.App.Save(lampaUser); err != nil {
			m.Logger.Error(
				"Failed to save Lampa user for new user",
				"UserId", user.Id,
				"Error", err,
			)
		}

		m.Logger.Info(
			"Created Lampa user for new user",
			"UserId", user.Id,
			"LampaUserId", lampaUser.Id,
		)

		return e.Next()
	})

	// User updated
	m.Ctx.App.OnRecordAfterUpdateSuccess("users").BindFunc(func(e *core.RecordEvent) error {
		user := users.ProxyUser(e.Record)
		needToSave := false
		lampaUser, err := m.GetLampaUserByUserId(user.Id)

		// New user
		if err != nil {
			if user.Role() == models.RoleGuest {
				return e.Next()
			}

			lampaUser, err = m.NewLampaUser(user)
			needToSave = true
			if err != nil {
				m.Logger.Error(
					"Failed to create Lampa user for updated user",
					"UserId", user.Id,
					"Error", err,
				)
				return e.Next()
			}
		}

		// Has changed role
		disabled := user.Role() == models.RoleGuest
		if lampaUser.Disabled() != disabled {
			lampaUser.SetDisabled(disabled)
			needToSave = true
		}

		// Has changes
		if needToSave {
			if err := m.Ctx.App.Save(lampaUser); err != nil {
				m.Logger.Error(
					"Failed to save Lampa user",
					"UserId", user.Id,
					"LampaUserId", lampaUser.Id,
					"Error", err,
				)
			}
		}

		return e.Next()
	})
}
