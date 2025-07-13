package outline

import (
	"time"

	"github.com/docker-pet/backend/models"
	"github.com/docker-pet/backend/modules/users"
	"github.com/pocketbase/pocketbase/core"
	"github.com/zmwangx/debounce"
)

func (m *OutlineModule) watchUsersChanges() {
	configureAll, _ := debounce.Debounce(
		func() { m.configureAll() },
		2*time.Second, // TODO: make configurable
		debounce.WithLeading(true),
		debounce.WithTrailing(true),
	)

	// On app start
	m.Ctx.App.OnServe().BindFunc(func(e *core.ServeEvent) error {
		configureAll()
		return e.Next()
	})

	// User created
	m.Ctx.App.OnRecordAfterCreateSuccess("users").BindFunc(func(e *core.RecordEvent) error {
		user := users.ProxyUser(e.Record)
		if user.Role() == models.RoleGuest {
			return e.Next()
		}

		configureAll()
		return e.Next()
	})

	// User updated
	m.Ctx.App.OnRecordAfterUpdateSuccess("users").BindFunc(func(e *core.RecordEvent) error {
		configureAll()
		return e.Next()
	})

	// User deleted
	m.Ctx.App.OnRecordDelete("users").BindFunc(func(e *core.RecordEvent) error {
		user := users.ProxyUser(e.Record)
		if user.Role() == models.RoleGuest {
			return e.Next()
		}

		configureAll()
		return e.Next()
	})
}
