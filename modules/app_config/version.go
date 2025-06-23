package app_config

import (
	"github.com/pocketbase/pocketbase/core"
)

func (m *AppConfigModule) setupVersionSync() {
	m.Ctx.App.OnServe().BindFunc(func(e *core.ServeEvent) error {
		config := m.AppConfig()
		l := config.Version()
		r := m.Config.Version

		if l == nil || l.Version != r.Version || l.Commit != r.Commit || l.BuildTime != r.BuildTime {
			m.Logger.Info("App version changed", "old", l, "new", r)
			config.SetVersion(&r)
			m.Ctx.App.Save(config)
		} else {
			m.Logger.Debug("App version unchanged", "version", l.Version)
		}

		return e.Next()
	})

}
