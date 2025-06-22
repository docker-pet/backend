package outline

import (
	"context"
	"io"
	"net"
	"net/http"
	"path"
	"path/filepath"

	"github.com/docker-pet/backend/models"
	"github.com/pocketbase/pocketbase/core"
	"resty.dev/v3"
)

func (m *OutlineModule) registerMetrixProxyEndpoint() {
	m.Ctx.App.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/api/outline/metics/{serverId}/{metricsSecret}", func(e *core.RequestEvent) error {
			// Get server
			serverId := e.Request.PathValue("serverId")
			metricsSecret := e.Request.PathValue("metricsSecret")
			server, err := m.GetServerById(serverId)

			if err != nil || server.MetricsSecret() != metricsSecret {
				return e.NotFoundError(
					"Server not found",
					"The server with the specified ID does not exist or the metrics secret is invalid.",
				)
			}

			if !server.Enabled() {
				return e.ForbiddenError(
					"Server is disabled",
					"The server is currently disabled and cannot be accessed.",
				)
			}

			var response *resty.Response

			// Request (Remote Sync)
			if server.SyncType() == models.OutlineRemoteSync {
				request, requestUrl := m.CreateCaddyRequest(server, "metrics")
				response, err = request.Get(requestUrl)
			} else if server.SyncType() == models.OutlineLocalSync {
				transport := &http.Transport{
					DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
						return net.Dial("unix", filepath.Join(m.Config.OutlineStoragePath, server.Slug(), "admin.sock"))
					},
				}

				client := resty.New().
					SetTransport(transport).
					SetBaseURL("http://unix")

				response, err = client.R().Get(path.Join("/", "metrics"))
			}

			if err != nil {
				m.Logger.Error(
					"Failed to fetch metrics",
					"Error", err,
					"ServerId", server.Id,
				)
				return e.InternalServerError(
					"Metrics fetch error",
					"An error occurred while fetching the metrics from the server.",
				)
			}

			// Resonse
			defer response.Body.Close()
			e.Response.Header().Set("Content-Type", response.Header().Get("Content-Type"))
			e.Response.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			e.Response.WriteHeader(response.StatusCode())
			_, err = io.Copy(e.Response, response.Body)
			return err
		})

		return se.Next()
	})
}
