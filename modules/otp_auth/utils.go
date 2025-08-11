package otp_auth

import (
	"net"

	"github.com/pocketbase/pocketbase/core"
)

func (m *OtpAuthModule) getAppDomain(e *core.RequestEvent) string {
	requestDomain, _, err := net.SplitHostPort(e.Request.Host)
	if err != nil {
		requestDomain = e.Request.Host
	}
	appDomain := m.appConfig.AppConfig().AppDomain()
	reverseDomain := m.appConfig.AppConfig().AppDomainReverse()
	if reverseDomain != "" && (requestDomain == reverseDomain ||
		(len(requestDomain) > len(reverseDomain)+1 &&
			requestDomain[len(requestDomain)-len(reverseDomain)-1:] == "."+reverseDomain)) {
		return reverseDomain
	}
	return appDomain
}
