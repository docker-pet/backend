package migrations

import (
	"fmt"

	"github.com/docker-pet/backend/models"
	"github.com/docker-pet/backend/modules/outline"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Collection
		collection, err := app.FindCollectionByNameOrId("outline_servers")
		if err != nil {
			return err
		}

		// Fields
		collection.Fields.Add(
			&core.SelectField{
				Name:      "syncType",
				Required:  true,
				Hidden:    true,
				Values:    []string{"local", "remote"},
				MaxSelect: 1,
			},
			&core.JSONField{
				Name:     "syncConfig",
				Required: false,
				Hidden:   true,
			},
		)

		if err := app.Save(collection); err != nil {
			return err
		}

		// Set default value for outline_token field
		records, err := app.FindAllRecords(collection)
		if err != nil {
			return err
		}
		for _, record := range records {
			server := outline.ProxyOutlineServer(record)
			if server == nil {
				continue
			}

			domain := "FILL_ME"

			outlineConfig := &models.OutlineServerSyncRemote{
				RemoteAdminEndpoint: fmt.Sprintf("https://%s/%s/", domain, server.GetString("servicePath")),
				RemoteAdminBasicAuth: &models.OutlineServerBasicAuth{
					Username: "service",
					Password: server.GetString("servicePassword"),
				},
				RemoteSyncCronExpression: "*/5 * * * *",
				OutlineConfig: &models.OutlineConfiguration{
					TCP: &models.OutlineConfigurationProtocol{
						Port:   server.GetInt("tcpPort"),
						Path:   server.GetString("tcpPath"),
						Prefix: server.GetString("tcpPrefix"),
					},
					UDP: &models.OutlineConfigurationProtocol{
						Port:   server.GetInt("udpPort"),
						Path:   server.GetString("udpPath"),
						Prefix: server.GetString("udpPrefix"),
					},
				},
			}

			server.SetSyncType(models.OutlineRemoteSync)
			server.SetSyncRemoteConfig(outlineConfig)

			if err := app.Save(record); err != nil {
				return err
			}
		}

		// Remove unused fields
		collection.Fields.RemoveByName("available")
		collection.Fields.RemoveByName("tcpPath")
		collection.Fields.RemoveByName("tcpPort")
		collection.Fields.RemoveByName("tcpPrefix")
		collection.Fields.RemoveByName("udpPath")
		collection.Fields.RemoveByName("udpPort")
		collection.Fields.RemoveByName("udpPrefix")
		collection.Fields.RemoveByName("servicePath")
		collection.Fields.RemoveByName("servicePassword")

		return app.Save(collection)
	}, nil)
}
