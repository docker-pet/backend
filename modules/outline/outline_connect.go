package outline

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/docker-pet/backend/models"
	"github.com/pocketbase/pocketbase/core"
	"gopkg.in/yaml.v3"
)

func (m *OutlineModule) registerOutlineConnectEndpoint() {
	m.Ctx.App.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.Any("/api/outline/{userId}/{outlineSecret}", func(e *core.RequestEvent) error {
			userId := e.Request.PathValue("userId")
			outlineSecret := e.Request.PathValue("outlineSecret")

			// Validate user
			user, err := m.users.GetUserById(userId)
			if err != nil {
				return e.NotFoundError(
					"User not found",
					"The user with the specified ID does not exist.",
				)
			}

			// Validate Outline Secret
			if user.OutlineToken() != outlineSecret {
				return e.UnauthorizedError(
					"Invalid Outline Secret",
					"The provided Outline secret is invalid or does not match the user's secret.",
				)
			}

			// Guest
			if !user.IsActive() {
				return e.UnauthorizedError(
					"Guest Access Denied",
					"Guests are not allowed to connect to Outline.",
				)
			}

			// TODO: server picking logic
			servers, err := m.GetAllActiveServers()
			if err != nil {
				return e.InternalServerError(
					"Failed to retrieve Outline servers",
					"An error occurred while trying to retrieve the list of active Outline servers.",
				)
			}

			// Available servers
			availableServers := make([]*models.OutlineServer, 0)
			for _, server := range servers {
				if !user.Premium() && server.Premium() {
					continue
				}
				availableServers = append(availableServers, server)
			}

			if len(servers) == 0 {
				return e.NotFoundError(
					"No Active Outline Servers",
					"There are currently no active Outline servers available for connection.",
				)
			}

			// Picked server
			rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
			server := availableServers[rnd.Intn(len(availableServers))]

			// Token
			token, err := m.tokenStore.GetOrGenerate(user.Id, server.Id)
			if err != nil {
				return e.InternalServerError(
					"Failed to generate Outline token",
					"An error occurred while trying to generate an Outline token for the user.",
				)
			}

			// Response
			root := &yaml.Node{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "transport"},
					{
						Kind: yaml.MappingNode,
						Content: []*yaml.Node{
							{Kind: yaml.ScalarNode, Value: "$type"},
							{Kind: yaml.ScalarNode, Value: "tcpudp"},

							{Kind: yaml.ScalarNode, Value: "tcp"},
							{
								Kind: yaml.MappingNode,
								Content: []*yaml.Node{
									{Kind: yaml.ScalarNode, Value: "$type"},
									{Kind: yaml.ScalarNode, Value: "shadowsocks"},

									{Kind: yaml.ScalarNode, Value: "endpoint"},
									{
										Kind: yaml.MappingNode,
										Content: []*yaml.Node{
											{Kind: yaml.ScalarNode, Value: "$type"},
											{Kind: yaml.ScalarNode, Value: "websocket"},
											{Kind: yaml.ScalarNode, Value: "url"},
											{Kind: yaml.ScalarNode, Value: fmt.Sprintf("wss://%s/%s", m.formatJobDomain(server), server.TCPPath())},
										},
									},
									{Kind: yaml.ScalarNode, Value: "cipher"},
									{Kind: yaml.ScalarNode, Value: m.Config.OutlineCipher},
									{Kind: yaml.ScalarNode, Value: "secret"},
									{Kind: yaml.ScalarNode, Value: token.Token},
								},
							},

							{Kind: yaml.ScalarNode, Value: "udp"},
							{
								Kind: yaml.MappingNode,
								Content: []*yaml.Node{
									{Kind: yaml.ScalarNode, Value: "$type"},
									{Kind: yaml.ScalarNode, Value: "shadowsocks"},

									{Kind: yaml.ScalarNode, Value: "endpoint"},
									{
										Kind: yaml.MappingNode,
										Content: []*yaml.Node{
											{Kind: yaml.ScalarNode, Value: "$type"},
											{Kind: yaml.ScalarNode, Value: "websocket"},
											{Kind: yaml.ScalarNode, Value: "url"},
											{Kind: yaml.ScalarNode, Value: fmt.Sprintf("wss://%s/%s", m.formatJobDomain(server), server.UDPPath())},
										},
									},
									{Kind: yaml.ScalarNode, Value: "cipher"},
									{Kind: yaml.ScalarNode, Value: m.Config.OutlineCipher},
									{Kind: yaml.ScalarNode, Value: "secret"},
									{Kind: yaml.ScalarNode, Value: token.Token},
								},
							},
						},
					},
				},
			}

			// Marshal YAML
			content, err := yaml.Marshal(root)
			if err != nil {
				m.Logger.Error("Failed to marshal Outline configuration", "Error", err)
				return e.InternalServerError(
					"Config Generation Error",
					"An error occurred while generating the Outline configuration.",
				)
			}

			// Response
			e.Response.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			return e.Blob(http.StatusOK, "application/yaml", content)
		})

		return se.Next()
	})
}
