package outline

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
)

func (m *OutlineModule) watchConfigChanges() {
	rebuildConfigs := func() {
		m.buildPrometheusConfig()
		m.buildPrometheusTargets()
		m.buildDockerComposeConfigs()
	}

	registerCronJobs := func(serverId string) {
		m.Ctx.App.Cron().MustAdd(generateCronJobName(serverId, "caddy"), m.Config.CaddyRefreshCronExpression, func() {
			m.configureCaddy(serverId)
		})
	}

	unregisterCronJobs := func(serverId string) {
		m.Ctx.App.Cron().Remove(generateCronJobName(serverId, "caddy"))
	}

	// On serve
	m.Ctx.App.OnServe().BindFunc(func(e *core.ServeEvent) error {
		rebuildConfigs()

		servers, err := m.GetAllServers()
		if err != nil {
			return fmt.Errorf("failed to get servers: %w", err)
		}
		for _, server := range servers {
			registerCronJobs(server.Id)
			m.configureCaddy(server.Id)
		}

		return e.Next()
	})

	// Delete
	m.Ctx.App.OnRecordAfterDeleteSuccess("outline_servers").BindFunc(func(e *core.RecordEvent) error {
		rebuildConfigs()
		unregisterCronJobs(e.Record.Id)
		return e.Next()
	})

	// Before create
	m.Ctx.App.OnRecordCreate("outline_servers").BindFunc(func(e *core.RecordEvent) error {
		outlineServer := ProxyOutlineServer(e.Record)

		outlineServer.GenerateTCPPath()
		outlineServer.GenerateUDPPath()
		outlineServer.GenerateServicePath()
		outlineServer.GenerateServicePassword()
		outlineServer.GenerateMetricsSecret()

		return e.Next()
	})

	// Create
	m.Ctx.App.OnRecordAfterCreateSuccess("outline_servers").BindFunc(func(e *core.RecordEvent) error {
		rebuildConfigs()
		registerCronJobs(e.Record.Id)
		return e.Next()
	})

	// Before update
	m.Ctx.App.OnRecordUpdate("outline_servers").BindFunc(func(e *core.RecordEvent) error {
		outlineServer := ProxyOutlineServer(e.Record)

		if outlineServer.TCPPath() == "" {
			outlineServer.GenerateTCPPath()
		}
		if outlineServer.UDPPath() == "" {
			outlineServer.GenerateUDPPath()
		}
		if outlineServer.ServicePath() == "" {
			outlineServer.GenerateServicePath()
		}
		if outlineServer.ServicePassword() == "" {
			outlineServer.GenerateServicePassword()
		}
		if outlineServer.MetricsSecret() == "" {
			outlineServer.GenerateMetricsSecret()
		}

		return e.Next()
	})

	// Update
	m.Ctx.App.OnRecordAfterUpdateSuccess("outline_servers").BindFunc(func(e *core.RecordEvent) error {
		rebuildConfigs()
		m.configureCaddy(e.Record.Id)
		return e.Next()
	})
}

func generateCronJobName(serverId string, postfix string) string {
	return "outline_" + postfix + ":" + serverId
}
