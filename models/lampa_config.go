package models

import (
	"github.com/Jeffail/gabs/v2"
	"github.com/pocketbase/pocketbase/core"
)

var _ core.RecordProxy = (*LampaConfig)(nil)

type LampaConfig struct {
	core.BaseRecordProxy
}

func (a *LampaConfig) DlnaEnabled() bool {
	return a.GetBool("dlnaEnabled")
}

func (a *LampaConfig) SetDlnaEnabled(value bool) {
	a.Set("dlnaEnabled", value)
}

func (a *LampaConfig) SisiEnabled() bool {
	return a.GetBool("sisiEnabled")
}

func (a *LampaConfig) SetSisiEnabled(value bool) {
	a.Set("sisiEnabled", value)
}

func (a *LampaConfig) TmdbProxyEnabled() bool {
	return a.GetBool("tmdbProxyEnabled")
}

func (a *LampaConfig) SetTmdbProxyEnabled(value bool) {
	a.Set("tmdbProxyEnabled", value)
}

func (a *LampaConfig) TorrServerEnabled() bool {
	return a.GetBool("torrServerEnabled")
}

func (a *LampaConfig) SetTorrServerEnabled(value bool) {
	a.Set("torrServerEnabled", value)
}

func (a *LampaConfig) ServerProxyEnabled() bool {
	return a.GetBool("serverProxyEnabled")
}

func (a *LampaConfig) SetServerProxyEnabled(value bool) {
	a.Set("serverProxyEnabled", value)
}

func (a *LampaConfig) CubEnabled() bool {
	return a.GetBool("cubEnabled")
}

func (a *LampaConfig) SetCubEnabled(value bool) {
	a.Set("cubEnabled", value)
}

func (a *LampaConfig) TracksEnabled() bool {
	return a.GetBool("tracksEnabled")
}

func (a *LampaConfig) SetTracksEnabled(value bool) {
	a.Set("tracksEnabled", value)
}

func (a *LampaConfig) OnlineEnabled() bool {
	return a.GetBool("onlineEnabled")
}

func (a *LampaConfig) SetOnlineEnabled(value bool) {
	a.Set("onlineEnabled", value)
}

func (a *LampaConfig) AdminPassword() string {
	return a.GetString("adminPassword")
}

func (a *LampaConfig) SetAdminPassword(value string) {
	a.Set("adminPassword", value)
}

func (a *LampaConfig) ConfigInit() *gabs.Container {
	value := a.GetString("configInit")
	if value == "" {
		return gabs.New()
	}

	jsonParsed, err := gabs.ParseJSON([]byte(value))
	if err != nil {
		panic(err)
	}

	return jsonParsed
}

func (a *LampaConfig) SetConfigInit(value *gabs.Container) {
	a.Set("configInit", value.StringIndent("", "  "))
}
