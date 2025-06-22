package lampa

import (
	"fmt"
	"path/filepath"

	"github.com/Jeffail/gabs/v2"
	"github.com/docker-pet/backend/helpers"
)

func (m *LampaModule) BuildInitConfig() {
	container := gabs.New()

	// Merge with custom lampa config
	err := container.Merge(m.LampaConfig().ConfigInit())
	if err != nil {
		m.Logger.Error("Failed to merge lampa config", "Err", err)
		panic(err)
	}

	// Listen
	container.SetP("https", "listenscheme")
	container.SetP(80, "listenport")
	container.SetP(fmt.Sprintf("lampa.%s", m.appConfig.AppConfig().AppDomain()), "listenhost")

	// Flags
	container.SetP(true, "multiaccess")
	container.SetP(false, "compression")

	// Browsers
	container.SetP(false, "chromium.enable")
	container.SetP(true, "firefox.enable")

	// DLNA
	container.SetP(m.LampaConfig().DlnaEnabled(), "dlna.enable")
	container.SetP(m.LampaConfig().DlnaEnabled(), "dlna.autoupdatetrackers")

	// LampaWeb
	container.SetP(m.LampaConfig().DlnaEnabled(), "LampaWeb.initPlugins.dlna")
	container.SetP(m.LampaConfig().TracksEnabled(), "LampaWeb.initPlugins.tracks")
	container.SetP(m.LampaConfig().TmdbProxyEnabled(), "LampaWeb.initPlugins.tmdbProxy")
	container.SetP(m.LampaConfig().OnlineEnabled(), "LampaWeb.initPlugins.online")
	container.SetP(m.LampaConfig().SisiEnabled(), "LampaWeb.initPlugins.sisi")
	container.SetP(m.LampaConfig().TorrServerEnabled(), "LampaWeb.initPlugins.torrserver")
	container.SetP(true, "LampaWeb.initPlugins.timecode")
	container.SetP(true, "LampaWeb.initPlugins.backup")
	container.SetP(true, "LampaWeb.initPlugins.sync")

	// TMDB
	container.SetP(false, "tmdb.enable")
	container.SetP(true, "tmdb.useproxy")
	container.SetP(true, "tmdb.useproxystream")

	// Cub
	container.SetP(m.LampaConfig().CubEnabled(), "cub.enable")

	// Server Proxy
	container.SetP(m.LampaConfig().ServerProxyEnabled(), "serverproxy.enable")
	container.SetP(false, "serverproxy.verifyip")
	container.SetP(true, "serverproxy.allow_tmdb")
	container.SetP(true, "serverproxy.image.cache")
	container.SetP(true, "serverproxy.image.cache_rsize")
	container.SetP(true, "serverproxy.buffering.enable")
	container.SetP(8192, "serverproxy.buffering.rent")
	container.SetP(3906, "serverproxy.buffering.length")
	container.SetP(5, "serverproxy.buffering.millisecondsTimeout")

	// AccessDB
	users, err := m.GetAllLampaUsers()
	if err != nil {
		m.Logger.Error("Failed to get all lampa users", "Err", err)
		panic(err)
	}

	container.SetP(true, "accsdb.enable")
	container.ArrayP("accsdb.users")
	for _, user := range users {
		userObj := gabs.New()
		userObj.SetP(user.AuthKey(), "id")
		userObj.SetP(user.Disabled(), "ban")
		userObj.SetP("2040-01-01T00:00:00", "expires")
		container.ArrayAppendP(userObj, "accsdb.users")
	}

	// Save to json
	updated, err := helpers.WriteFileIfChanged(
		filepath.Join(m.Config.StoragePath, "init.conf"),
		[]byte(container.StringIndent("", "  ")),
		0644,
	)

	// Log error if writing failed
	if err != nil {
		m.Logger.Error("Failed to write init.conf", "Err", err)
		return
	} else if updated {
		m.Logger.Info("Init.conf updated successfully")
	}
}
