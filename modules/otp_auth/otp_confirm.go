package otp_auth

import (
	"net/http"

	"github.com/Jeffail/gabs/v2"
	"github.com/docker-pet/backend/helpers"
	"github.com/docker-pet/backend/models"
	"github.com/docker-pet/backend/modules/users"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func (m *OtpAuthModule) registerOtpConfirmEndpoint() {
	m.Ctx.App.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.POST("/api/otp/confirm", func(e *core.RequestEvent) error {
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

			// Otp code
			otpCode, ok := data.Path("code").Data().(string)
			if !ok {
				return e.BadRequestError("field 'code' must be a string", nil)
			}

			// Not found
			if found := m.keychain.Exists(otpCode); !found {
				container := gabs.New()
				container.Set("not_found", "notification")
				return e.JSON(http.StatusOK, container.Data())
			}

			// Confirm auth
			m.keychain.Confirm(otpCode, user.Id, user.Role())
			container := gabs.New()
			container.Set("confirmed", "notification")
			return e.JSON(http.StatusOK, container.Data())
		}).Bind(apis.RequireAuth("users"))

		return se.Next()
	})
}
