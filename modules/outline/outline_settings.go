package outline

import (
	"net/http"

	"github.com/docker-pet/backend/helpers"
	"github.com/docker-pet/backend/models"
	"github.com/docker-pet/backend/modules/users"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func (m *OutlineModule) registerSettingsEndpoint() {
	m.Ctx.App.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.POST("/api/outline/settings", func(e *core.RequestEvent) error {
			// User
			user := users.ProxyUser(e.Auth)
			if user.Role() == models.RoleGuest {
				return e.UnauthorizedError("Guest users are not allowed to confirm OTP", user)
			}

			// Parse JSON body
			data, err := helpers.ParseJSONBodyLimited(e.Request.Body)
			if err != nil {
				return e.BadRequestError(err.Error(), nil)
			}

			// Prefix enabled
			outlinePrefixEnabled, ok := data.Path("outlinePrefixEnabled").Data().(bool)
			if !ok {
				return e.BadRequestError("field 'outlinePrefixEnabled' must be a bool", nil)
			}

			// Picked server
			outlineServerId, ok := data.Path("outlineServer").Data().(string)
			if !ok {
				return e.BadRequestError("field 'outlineServer' must be a string", nil)
			}

			// Check server
			// TODO: chech if server is active
			if outlineServerId != "" {
				_, err := m.GetServerById(outlineServerId)
				if err != nil {
					return e.BadRequestError("server with specified ID not found", nil)
				}
			}

			// Save settings
			user.SetOutlinePrefixEnabled(outlinePrefixEnabled)
			user.SetOutlineServer(outlineServerId)
			if err := m.Ctx.App.Save(user); err != nil {
				return e.InternalServerError("Failed to save user settings", err)
			}

			return e.JSON(http.StatusOK, "Settings updated successfully")
		}).Bind(apis.RequireAuth("users"))

		return se.Next()
	})
}
