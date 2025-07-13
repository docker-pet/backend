package otp_auth

import (
	"net/http"

	"github.com/Jeffail/gabs/v2"
	"github.com/pocketbase/pocketbase/core"
)

func (m *OtpAuthModule) registerOtpUserEndpoint() {
	m.Ctx.App.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.Any("/api/otp/me", func(e *core.RequestEvent) error {
			claims := m.parseCooke(e)

			// Unauthenticated user
			if claims.UserId == "" {
				return e.UnauthorizedError("You are not authenticated. Please login first.", nil)
			}

			// Response
			container := gabs.New()
			container.Set(claims.UserId, "userId")
			container.Set(claims.UserRole, "role")
			container.Set(claims.DeviceName, "deviceName")

			// Lampa
			withLampa := e.Request.URL.Query().Has("with-lampa")
			if withLampa {
				if lampaUser, _ := m.lampa.GetLampaUserByUserId(claims.UserId); lampaUser != nil {
					container.Set(lampaUser.AuthKey(), "lampaAuthKey")
				}
			}

			return e.JSON(http.StatusOK, container.Data())
		})

		return se.Next()
	})
}
