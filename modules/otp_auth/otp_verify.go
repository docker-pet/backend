package otp_auth

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/pocketbase/pocketbase/core"
)

func (m *OtpAuthModule) registerOtpVerifyEndpoint() {
	m.Ctx.App.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.Any("/api/otp/verify", func(e *core.RequestEvent) error {
			claims := m.parseCooke(e)

			// Already authenticated
			if claims.UserId != "" {
				return e.NoContent(http.StatusOK)
			}

			// Redirect to auth page
			redirectDomain := m.getAppDomain(e)
			redirectUrl := "https://" + redirectDomain
			if e.Request.Header.Get("Remote-Addr") != "" {
				redirectUrl = "https://" + e.Request.Header.Get("Remote-Addr") + e.Request.Header.Get("Original-URI")
			}

			return e.Redirect(302, fmt.Sprintf(
				"https://%s/auth?redirect=%s",
				redirectDomain,
				url.QueryEscape(redirectUrl),
			))
		})

		return se.Next()
	})
}
