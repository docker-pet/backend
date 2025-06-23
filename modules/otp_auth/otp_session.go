package otp_auth

import (
	"net/http"
	"time"

	"github.com/Jeffail/gabs/v2"
	"github.com/docker-pet/backend/helpers"
	"github.com/pocketbase/pocketbase/core"
)

func (m *OtpAuthModule) registerOtpSessionEndpoint() {
	m.Ctx.App.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.POST("/api/otp/session", func(e *core.RequestEvent) error {
			claims := m.parseCooke(e)

			// Parse JSON body
			data, err := helpers.ParseJSONBodyLimited(e.Request.Body)
			if err != nil {
				return e.BadRequestError(err.Error(), nil)
			}

			// Device name is required
			// TODO: Device name length limit to config file
			deviceName, ok := data.Path("deviceName").Data().(string)
			if !ok || len(claims.DeviceName) > 86 {
				return e.BadRequestError("field 'deviceName' must be a string and not longer than 86 characters", nil)
			}
			claims.DeviceName = deviceName

			// Already authenticated
			if claims.UserId != "" {
				container := gabs.New()
				container.Set(claims.UserId, "userId")
				container.Set(claims.DeviceName, "deviceName")
				return e.JSON(http.StatusOK, container.Data())
			}

			// Generate unique OTP code
			if claims.Pin == "" {
				reserved := false
				for i := 0; i < m.Config.MaxPinGenerationAttempts; i++ {
					pin, err := helpers.GeneratePinCode(m.appConfig.AppConfig().AuthPinLength())
					if err != nil {
						return e.InternalServerError("Failed to generate PIN code", err)
					}
					if reserved = m.keychain.Reserve(pin); reserved {
						claims.Pin = pin
						break
					}
				}

				if !reserved {
					return e.InternalServerError("Failed to generate PIN code after multiple attempts", nil)
				}

				m.fillCookie(e, *claims)
			}

			// Session confirmed
			if keychainUser, confirmed := m.keychain.IsConfirmed(claims.Pin); confirmed {
				claims.Pin = ""
				claims.UserId = keychainUser.UserId
				claims.UserRole = keychainUser.UserRole
				claims.ValidationDate = time.Now()
				m.fillCookie(e, *claims)

				container := gabs.New()
				container.Set(claims.UserId, "userId")
				container.Set(claims.DeviceName, "deviceName")
				return e.JSON(http.StatusOK, container.Data())
			}

			// Response
			container := gabs.New()
			container.Set(claims.Pin, "pin")
			container.Set(claims.DeviceName, "deviceName")

			return e.JSON(http.StatusOK, container.Data())
		})

		return se.Next()
	})
}
