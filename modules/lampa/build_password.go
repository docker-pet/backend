package lampa

import (
	"path/filepath"

	"github.com/docker-pet/backend/helpers"
)

func (m *LampaModule) BuildPassword() {
	// Save to passwd file
	updated, err := helpers.WriteFileIfChanged(
		filepath.Join(m.Config.StoragePath, "passwd"),
		[]byte(m.LampaConfig().AdminPassword()),
		0644,
	)

	// Log error if writing failed
	if err != nil {
		m.Logger.Error("Failed to write passwd", "Err", err)
		return
	} else if updated {
		m.Logger.Info("Admin password updated successfully")
	}
}
