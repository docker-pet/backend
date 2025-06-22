package outline

import (
	"fmt"

	"github.com/docker-pet/backend/models"
	"github.com/pocketbase/pocketbase/core"
)

func (m *OutlineModule) watchConfigChanges() {
	rebuildConfigs := func() {
		m.buildPrometheusConfig()
		m.buildPrometheusTargets()
		m.buildDockerComposeConfigs()
	}

	rebuildCronJobs := func() {
		servers, err := m.GetAllServers()
		if err != nil {
			m.Logger.Error("Failed to get servers for unregistering cron jobs", "error", err)
			return
		}

		// Remove existing cron jobs
		cronJobs := m.Ctx.App.Cron().Jobs()
		for _, job := range cronJobs {
			if len(job.Id()) >= 8 && job.Id()[:8] == "outline_" {
				m.Ctx.App.Cron().Remove(job.Id())
			}
		}

		// Register new cron jobs
		for _, server := range servers {
			cronExpression := "0 0 * * *" // Default cron expression for Caddy refresh

			if server.SyncType() != models.OutlineLocalSync {
				config := server.SyncRemoteConfig()
				if config.RemoteSyncCronExpression != "" {
					cronExpression = config.RemoteSyncCronExpression
				}
			}

			m.Ctx.App.Cron().MustAdd(generateCronJobName(server.Slug()), cronExpression, func() {
				m.configureCaddy(server.Id)
			})
		}
	}

	// On serve
	m.Ctx.App.OnServe().BindFunc(func(e *core.ServeEvent) error {
		go rebuildConfigs()
		rebuildCronJobs()

		servers, err := m.GetAllServers()
		if err != nil {
			return fmt.Errorf("failed to get servers: %w", err)
		}
		for _, server := range servers {
			go m.configureCaddy(server.Id)
		}

		return e.Next()
	})

	// Delete
	m.Ctx.App.OnRecordAfterDeleteSuccess("outline_servers").BindFunc(func(e *core.RecordEvent) error {
		go rebuildConfigs()
		rebuildCronJobs()
		return e.Next()
	})

	// Before create
	m.Ctx.App.OnRecordCreate("outline_servers").BindFunc(func(e *core.RecordEvent) error {
		outlineServer := ProxyOutlineServer(e.Record)
		outlineServer.GenerateMetricsSecret()
		if outlineServer.SyncType() == models.OutlineLocalSync {
			outlineServer.SetSyncLocalConfig(outlineServer.SyncLocalConfig())
		} else {
			outlineServer.SetSyncRemoteConfig(outlineServer.SyncRemoteConfig())
		}
		return e.Next()
	})

	// Create
	m.Ctx.App.OnRecordAfterCreateSuccess("outline_servers").BindFunc(func(e *core.RecordEvent) error {
		outlineServer := ProxyOutlineServer(e.Record)
		go rebuildConfigs()
		rebuildCronJobs()
		m.configureCaddy(outlineServer.Id)
		return e.Next()
	})

	// Before update
	m.Ctx.App.OnRecordUpdate("outline_servers").BindFunc(func(e *core.RecordEvent) error {
		outlineServer := ProxyOutlineServer(e.Record)
		outlineServer.GenerateMetricsSecret()
		if outlineServer.SyncType() == models.OutlineLocalSync {
			outlineServer.SetSyncLocalConfig(outlineServer.SyncLocalConfig())
		} else {
			outlineServer.SetSyncRemoteConfig(outlineServer.SyncRemoteConfig())
		}

		if outlineServer.MetricsSecret() == "" {
			outlineServer.GenerateMetricsSecret()
		}

		return e.Next()
	})

	// Update
	m.Ctx.App.OnRecordAfterUpdateSuccess("outline_servers").BindFunc(func(e *core.RecordEvent) error {
		outlineServer := ProxyOutlineServer(e.Record)
		go rebuildConfigs()
		go m.configureCaddy(outlineServer.Id)
		rebuildCronJobs()
		return e.Next()
	})
}

func generateCronJobName(serverSlug string) string {
	return "outline_" + serverSlug
}
