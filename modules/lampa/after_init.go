package lampa

import (
	"path/filepath"

	"github.com/docker-pet/backend/helpers"
	"github.com/pocketbase/pocketbase/core"
)

func (m *LampaModule) afterInit() {
	m.Ctx.App.OnServe().BindFunc(func(e *core.ServeEvent) error {
		cleanPath := filepath.Clean(m.Config.StoragePath)
		err := helpers.EnsureDir(cleanPath)

		if err != nil {
			m.Logger.Error(
				"Failed to ensure lampa configs directory",
				"Error", err,
				"Path", cleanPath,
			)
			return err
		}

		m.BuildManifest()
		m.BuildPassword()
		m.BuildInitConfig()

		return e.Next()
	})
}
