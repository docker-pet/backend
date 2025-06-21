package models

import (
	"github.com/biter777/countries"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/security"
)

var _ core.RecordProxy = (*OutlineServer)(nil)

type OutlineServer struct {
	core.BaseRecordProxy
}

func (a *OutlineServer) Slug() string {
	return a.GetString("slug")
}

func (a *OutlineServer) SetSlug(value string) {
	a.Set("slug", value)
}

func (a *OutlineServer) Country() countries.CountryCode {
	return countries.ByName(a.GetString("country"))
}

func (a *OutlineServer) SetCountry(value string) {
	a.Set("country", value)
}

func (a *OutlineServer) Enabled() bool {
	return a.GetBool("enabled")
}

func (a *OutlineServer) Available() bool {
	return a.GetBool("available")
}

func (a *OutlineServer) SetAvailable(value bool) {
	a.Set("available", value)
}

func (a *OutlineServer) SetEnabled(value bool) {
	a.Set("enabled", value)
}

func (a *OutlineServer) Premium() bool {
	return a.GetBool("premium")
}

func (a *OutlineServer) SetPremium(value bool) {
	a.Set("premium", value)
}

func (a *OutlineServer) OverrideDomain() string {
	return a.GetString("overrideDomain")
}

func (a *OutlineServer) SetOverrideDomain(value string) {
	a.Set("overrideDomain", value)
}

func (a *OutlineServer) TCPPath() string {
	return a.GetString("tcpPath")
}

func (a *OutlineServer) GenerateTCPPath() {
	a.Set("tcpPath", security.RandomString(32))
}

func (a *OutlineServer) UDPPath() string {
	return a.GetString("udpPath")
}

func (a *OutlineServer) GenerateUDPPath() {
	a.Set("udpPath", security.RandomString(32))
}

func (a *OutlineServer) ServicePath() string {
	return a.GetString("servicePath")
}

func (a *OutlineServer) GenerateServicePath() {
	a.Set("servicePath", security.RandomString(32))
}

func (a *OutlineServer) ServicePassword() string {
	return a.GetString("servicePassword")
}

func (a *OutlineServer) GenerateServicePassword() {
	a.Set("servicePassword", security.RandomString(32))
}

func (a *OutlineServer) MetricsSecret() string {
	return a.GetString("metricsSecret")
}

func (a *OutlineServer) GenerateMetricsSecret() {
	a.Set("metricsSecret", security.RandomString(32))
}
