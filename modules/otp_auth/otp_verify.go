package otp_auth

import (
	"fmt"
	"net"
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

			// Request domain
			requestDomain, _, err := net.SplitHostPort(e.Request.Host)
			if err != nil {
				requestDomain = e.Request.Host
			}

			// App Domain (main or reverse)
			appDomain := m.appConfig.AppConfig().AppDomain()
			reverseDomain := m.appConfig.AppConfig().AppDomainReverse()
			redirectDomain := appDomain
			if reverseDomain != "" && (requestDomain == reverseDomain ||
				(len(requestDomain) > len(reverseDomain)+1 &&
					requestDomain[len(requestDomain)-len(reverseDomain)-1:] == "."+reverseDomain)) {
				redirectDomain = reverseDomain
			}

			// Redirect to auth page
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
