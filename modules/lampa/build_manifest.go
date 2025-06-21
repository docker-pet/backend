package lampa

import (
	"path/filepath"

	"github.com/Jeffail/gabs/v2"
	"github.com/docker-pet/backend/helpers"
)

func (m *LampaModule) BuildManifest() {
	container := gabs.New()
	container.Array()

	// Modules
	appendManifest(container, "Online.dll", m.LampaConfig().OnlineEnabled(), "")
	appendManifest(container, "SISI.dll", m.LampaConfig().SisiEnabled(), "")
	appendManifest(container, "DLNA.dll", m.LampaConfig().DlnaEnabled(), "")
	appendManifest(container, "Tracks.dll", m.LampaConfig().TracksEnabled(), "Tracks.ModInit")
	appendManifest(container, "TorrServer.dll", m.LampaConfig().TorrServerEnabled(), "TorrServer.ModInit")

	// Save to json
	updated, err := helpers.WriteFileIfChanged(
		filepath.Join(m.Config.StoragePath, "manifest.json"),
		[]byte(container.StringIndent("", "  ")),
		0644,
	)

	// Log error if writing failed
	if err != nil {
		m.Logger.Error("Failed to write manifest.json", "Err", err)
		return
	} else if updated {
		m.Logger.Info("Manifest.json updated successfully")
	}
}

func appendManifest(container *gabs.Container, name string, enable bool, initspace string) {
	obj := gabs.New()
	obj.Set(name, "dll")
	obj.Set(enable, "enable")
	if initspace != "" {
		obj.Set(initspace, "initspace")
	}
	container.ArrayAppend(obj)
}
