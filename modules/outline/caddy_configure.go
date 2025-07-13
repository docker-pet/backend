package outline

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/Jeffail/gabs/v2"
	"github.com/docker-pet/backend/helpers"
	"github.com/docker-pet/backend/models"
	"github.com/pocketbase/pocketbase/tools/security"
	"golang.org/x/crypto/bcrypt"
	"resty.dev/v3"
)

type Token struct {
	UserId string
	Token  string
}

func (m *OutlineModule) configureAll() {
	servers, err := m.GetAllActiveServers()
	if err != nil {
		m.Logger.Warn(
			"Failed to get all active Outline servers",
			"Error", err,
		)
		return
	}

	for _, server := range servers {
		m.configureCaddy(server.Id)
	}
}

func (m *OutlineModule) configureCaddy(serverId string) {
	// Tokens
	var tokens []*Token
	if users, _ := m.users.GetAllUsers(); users != nil {
		for _, user := range users {
			if user.OutlineToken() != "" && user.IsActive() {
				tokens = append(tokens, &Token{
					UserId: user.Id,
					Token:  user.OutlineToken(),
				})
			}
		}
	}

	server, err := m.GetServerById(serverId)
	hasServerChanges := false
	if err != nil {
		m.Logger.Warn(
			"Failed to get server by ID",
			"Error", err,
			"ServerId", serverId,
		)
		return
	}

	configureJustLocalFile := server.SyncType() == models.OutlineLocalSync
	var rawConfig []byte
	if server.SyncType() == models.OutlineRemoteSync {
		rawConfig, err = m.getRemoteCaddyConfig(server)
	} else if server.SyncType() == models.OutlineLocalSync {
		rawConfig, hasServerChanges, err = m.getLocalCaddyConfig(server)
	}

	if err != nil {
		m.Logger.Warn(
			"Failed to get Caddy configuration",
			"Reason", err.Error(),
		)

		if server.SyncType() == models.OutlineRemoteSync {
			configureJustLocalFile = true
			rawConfig, hasServerChanges = m.GenerateBasicCaddyConfig(server)

			m.Logger.Info(
				"Using basic Caddy configuration",
			)
		} else {
			return
		}
	}

	// Need to save server changes & stop configure
	if hasServerChanges {
		m.Logger.Info(
			"Server configuration has changes, stopping Caddy configuration",
			"ServerId", server.Id,
			"ServerSlug", server.Slug(),
		)
		err = m.Ctx.App.Save(server)
		if err != nil {
			m.Logger.Warn(
				"Failed to save server changes",
				"Error", err,
				"ServerId", server.Id,
				"ServerSlug", server.Slug(),
			)
		}
		return
	}

	// Parse the Caddy configuration
	config, err := gabs.ParseJSON(rawConfig)
	if err != nil {
		println(string(rawConfig))

		m.Logger.Warn(
			"Failed to parse Caddy configuration",
			"Error", err,
			"ServerId", server.Id,
			"SyncType", server.SyncType(),
		)
		return
	}

	needToSave := m.hasKeysChanges(config, tokens)

	// clear websocket2layer4 routes
	err = removeWebsocket2Layer4Routes(config)
	if err != nil {
		m.Logger.Warn(
			"Failed to remove websocket2layer4 routes",
			"Error", err,
			"ServerId", server.Id,
			"Host", m.formatJobDomain(server),
		)
		return
	}

	// find server key
	serverKey, err := findServerKey(config, server)
	if err != nil {
		m.Logger.Warn(
			"Failed to find server key in Caddy configuration",
			"Error", err,
			"ServerId", server.Id,
			"Host", m.formatJobDomain(server),
		)
		return
	}

	// websocket2layer4
	if err = arrayPrepend(config, "apps.http.servers."+serverKey+".routes",
		m.generateWebsocket2layer4Route(server, "tcp"),
		m.generateWebsocket2layer4Route(server, "udp"),
	); err != nil {
		m.Logger.Warn(
			"Failed to prepend websocket2layer4 routes",
			"Error", err,
			"ServerId", server.Id,
			"Host", m.formatJobDomain(server),
		)
		return
	}

	// No changes needed
	if !needToSave {
		m.Logger.Info(
			"No changes needed for Outline configuration",
			"ServerId", server.Id,
			"TokensCount", len(tokens),
		)
		return
	}

	// Outline config
	config.SetP(m.generateOutlineConfig(server, tokens), "apps.outline")

	m.Logger.Info(
		"Saving Outline configuration",
		"ServerId", server.Id,
		"ServerSlug", server.Slug(),
		"TokensCount", len(tokens),
	)

	// Save the config to file
	configPath := m.GenerateCaddyConfigPath(server.Slug())
	helpers.EnsureDir(filepath.Dir(configPath))
	_, err = helpers.WriteFileIfChanged(
		configPath,
		[]byte(config.StringIndent("", "  ")),
		0644,
	)

	// Save remote Caddy configuration
	if server.SyncType() == models.OutlineRemoteSync && !configureJustLocalFile {
		err = m.saveRemoteCaddyConfig(server, config.String())
	}

	if err != nil {
		m.Logger.Warn(
			"Failed to save Caddy configuration",
			"Error", err,
			"ServerId", server.Id,
			"ServerSlug", server.Slug(),
		)
	}
}

func (m *OutlineModule) CreateCaddyRequest(server *models.OutlineServer, command string) (*resty.Request, string) {
	config := server.SyncRemoteConfig()

	endpoint := config.RemoteAdminEndpoint
	if endpoint == "" {
		endpoint = "/"
	}

	isFullURL := strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://")

	if !isFullURL {
		endpoint = strings.TrimRight(endpoint, "/")
		endpoint = fmt.Sprintf("https://%s%s", m.formatJobDomain(server), endpoint)
	}

	requestUrl := fmt.Sprintf("%s/%s", strings.TrimRight(endpoint, "/"), strings.TrimLeft(command, "/"))

	request := m.Ctx.HttpClient.R()
	if config.RemoteAdminBasicAuth != nil && config.RemoteAdminBasicAuth.Username != "" {
		request = request.SetBasicAuth(
			config.RemoteAdminBasicAuth.Username,
			config.RemoteAdminBasicAuth.Password,
		)
	}

	return request, requestUrl
}

func (m *OutlineModule) getRemoteCaddyConfig(server *models.OutlineServer) ([]byte, error) {
	request, requestUrl := m.CreateCaddyRequest(server, "config")
	response, err := request.Get(requestUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch remote Caddy config: %w", err)
	}

	if response.StatusCode() != 200 {
		return nil, fmt.Errorf("failed to fetch remote Caddy config: %s", response.Status())
	}

	return response.Bytes(), nil
}

func (m *OutlineModule) saveRemoteCaddyConfig(server *models.OutlineServer, config string) error {
	request, requestUrl := m.CreateCaddyRequest(server, "config")
	_, err := request.
		SetContentType("application/json").
		SetBody(config).
		Patch(requestUrl)
	return err
}

func (m *OutlineModule) getLocalCaddyConfig(server *models.OutlineServer) ([]byte, bool, error) {
	configPath := m.GenerateCaddyConfigPath(server.Slug())

	// File does not exist (using os stat)
	if _, err := os.Stat(configPath); err != nil || os.IsNotExist(err) {
		content, hasChanges := m.GenerateBasicCaddyConfig(server)
		return content, hasChanges, nil
	}

	// Get file content
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, false, fmt.Errorf("failed to read local Caddy config: %w", err)
	}
	return data, false, nil
}

func (m *OutlineModule) GenerateBasicCaddyConfig(server *models.OutlineServer) ([]byte, bool) {
	outlineConfig := server.OutlineConfig()
	ports := []int{outlineConfig.TCP.Port, outlineConfig.UDP.Port}
	portsJson := `":443", ":80"`
	for _, port := range ports {
		if port == 0 {
			continue // Skip zero ports
		}
		portsJson += fmt.Sprintf(`, ":%d"`, port)
	}

	switch server.SyncType() {
	case models.OutlineLocalSync:
		return []byte(fmt.Sprintf(`{
			"admin": {
				"listen": "unix//outline_generated/admin.sock"
			},
			"storage": {
				"module": "file_system",
				"root": "/config"
			},
			"apps": {
				"http": {
					"servers": {
						"srv0": {
							"listen": [%s],
							"routes": []
						}
					}
				},
				"tls": {
					"automation": {
						"on_demand": {}
					}
				}
			}
		}`, portsJson)), false
	case models.OutlineRemoteSync:
		config := server.SyncRemoteConfig()
		servicePath := helpers.ExtractUrlPath(config.RemoteAdminEndpoint)
		serviceUsername := config.RemoteAdminBasicAuth.Username
		servicePassword := config.RemoteAdminBasicAuth.Password
		needToSave := false

		if servicePath == "/" {
			needToSave = true
			config.RemoteAdminEndpoint = fmt.Sprintf("/%s", security.RandomString(32))
		}

		if serviceUsername == "" || servicePassword == "" {
			needToSave = true
			config.RemoteAdminBasicAuth.Username = "service"
			config.RemoteAdminBasicAuth.Password = security.RandomString(32)
		}

		// Hash the password using bcrypt
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(servicePassword), bcrypt.DefaultCost)
		if err == nil {
			servicePassword = string(hashedPassword)
		} else {
			servicePassword = "__FILL_ME__"
		}

		if needToSave {
			server.SetSyncRemoteConfig(config)
		}

		// Domains
		domains := fmt.Sprintf(`"%s"`, m.formatJobDomain(server))
		reverseDomain := m.formatReverseDomain(server)
		if reverseDomain != "" {
			domains += fmt.Sprintf(`, "%s"`, reverseDomain)
		}

		return []byte(fmt.Sprintf(`{
			"admin": {
              "listen": "unix//outline_generated/admin.sock"
            },
            "storage": {
              "module": "file_system",
              "root": "/config"
            },
            "apps": {
              "http": {
                "servers": {
                  "srv0": {
                    "listen": [%s],
                    "routes": [
                      {
                        "match": [
                          {
                            "host": [%s]
                          }
                        ],
                        "handle": [
                          {
                            "handler": "subroute",
                            "routes": [
                              {
                                "handle": [
                                  {
                                    "handler": "subroute",
                                    "routes": [
                                      {
                                        "handle": [
                                          {
                                            "handler": "rewrite",
                                            "strip_path_prefix": "%s"
                                          }
                                        ]
                                      },
                                      {
                                        "handle": [
                                          {
                                            "handler": "authentication",
                                            "providers": {
                                              "http_basic": {
                                                "accounts": [
                                                  {
                                                    "username": "%s",
                                                    "password": "%s"
                                                  }
                                                ],
                                                "hash": {
                                                  "algorithm": "bcrypt"
                                                },
                                                "hash_cache": {}
                                              }
                                            }
                                          },
                                          {
                                            "handler": "reverse_proxy",
                                            "upstreams": [
                                              {
                                                "dial": "unix//var/run/caddy/admin.sock"
                                              }
                                            ]
                                          }
                                        ]
                                      }
                                    ]
                                  }
                                ],
                                "match": [
                                  {
                                    "path": [
                                      "%s/*"
                                    ]
                                  }
                                ]
                              }
                            ]
                          }
                        ],
                        "terminal": true
                      }
                    ]
                  }
                }
              }
            }
          }`, portsJson, domains, servicePath, serviceUsername, servicePassword, servicePath)), needToSave
	}

	return []byte("{}"), false
}

func (m *OutlineModule) GenerateCaddyConfigPath(serverSlug string) string {
	return filepath.Join(m.Config.OutlineStoragePath, serverSlug, "caddy.config.json")
}

func (m *OutlineModule) generateKey(token *Token) *gabs.Container {
	key := gabs.New()
	key.SetP(m.Config.OutlineCipher, "cipher")
	key.SetP(token.UserId, "id")
	key.SetP(token.Token, "secret")
	return key
}

func (m *OutlineModule) generateOutlineConfig(server *models.OutlineServer, tokens []*Token) *gabs.Container {
	container := gabs.New()

	// Shadowsocks
	shadowsocks := gabs.New()
	shadowsocks.SetP(10000, "replay_history")
	container.SetP(shadowsocks, "shadowsocks")

	// Handle Container Array
	handleContainerArray := gabs.New()
	handleContainerArray.Array()
	container.SetP(handleContainerArray, "connection_handlers")

	// Handle Container
	handleContainer := gabs.New()
	handleContainer.SetP(server.Slug(), "name")
	handleContainerArray.ArrayAppend(handleContainer)

	// Handle
	handle := gabs.New()
	handle.SetP("shadowsocks", "handler")
	handleContainer.SetP(handle, "handle")

	// Keys
	keys := gabs.New()
	keys.Array()
	for _, token := range tokens {
		keys.ArrayAppend(m.generateKey(token))
	}
	handle.SetP(keys, "keys")

	return container
}

func (m *OutlineModule) generateWebsocket2layer4Route(server *models.OutlineServer, packageType string) *gabs.Container {
	route := gabs.New()
	config := server.OutlineConfig()

	handleType := "stream"
	serverPath := config.TCP.Path
	if packageType == "udp" {
		handleType = "packet"
		serverPath = config.UDP.Path
	}

	handle := gabs.New()
	handle.Array()
	handleItem := gabs.New()
	handleItem.Set(server.Slug(), "connection_handler")
	handleItem.Set("websocket2layer4", "handler")
	handleItem.Set(handleType, "type")
	handle.ArrayAppend(handleItem)
	route.SetP(handle, "handle")

	match := gabs.New()
	match.Array()
	route.SetP(match, "match")

	matchHost := gabs.New()
	matchHostItem := gabs.New()
	matchHostItem.Array()
	matchHostItem.ArrayAppend(m.formatJobDomain(server))
	reverseDomain := m.formatReverseDomain(server)
	if reverseDomain != "" {
		matchHostItem.ArrayAppend(reverseDomain)
	}
	matchHost.SetP(matchHostItem, "host")
	match.ArrayAppend(matchHost)

	matchPath := gabs.New()
	matchPathItem := gabs.New()
	matchPathItem.Array()
	matchPathItem.ArrayAppend(path.Join("/", serverPath))
	matchPath.SetP(matchPathItem, "path")
	match.ArrayAppend(matchPath)

	return route
}

func (m *OutlineModule) hasKeysChanges(config *gabs.Container, tokens []*Token) bool {
	// Collecting up-to-date data on tokens.
	tokenInfo := make(map[string]struct {
		userId string
		cipher string
	})
	for _, t := range tokens {
		tokenInfo[t.Token] = struct {
			userId string
			cipher string
		}{userId: t.UserId, cipher: m.Config.OutlineCipher}
	}

	// Current keys
	currentKeys := config.Path("apps.outline.connection_handlers.0.keys")
	needToSave := false

	if currentKeys != nil && currentKeys.Data() != nil {
		if arr, ok := currentKeys.Data().([]interface{}); ok {
			seen := make(map[string]bool)
			for _, k := range arr {
				keyObj, ok := k.(map[string]interface{})
				if !ok {
					needToSave = true
					break
				}
				secret, _ := keyObj["secret"].(string)
				userId, _ := keyObj["id"].(string)
				cipher, _ := keyObj["cipher"].(string)
				info, exists := tokenInfo[secret]
				if !exists || info.userId != userId || info.cipher != cipher {
					needToSave = true
					break
				}
				seen[secret] = true
			}
			// Проверяем, есть ли новые токены, которых нет в keys
			if !needToSave {
				for secret := range tokenInfo {
					if !seen[secret] {
						needToSave = true
						break
					}
				}
			}
		} else {
			needToSave = true
		}
	} else {
		needToSave = true
	}

	return needToSave
}

func removeWebsocket2Layer4Routes(root *gabs.Container) error {
	servers := root.Path("apps.http.servers")
	if servers == nil {
		return fmt.Errorf("path apps.http.servers not found")
	}

	// Iterate over each server
	for srvName, srv := range servers.ChildrenMap() {
		routesContainer := srv.S("routes")
		if routesContainer == nil {
			continue
		}

		// Extract raw slice of routes
		rawRoutes, ok := routesContainer.Data().([]interface{})
		if !ok {
			continue
		}

		var filtered []interface{}

		// Iterate and filter
		for i, raw := range rawRoutes {
			route := routesContainer.Children()[i]
			handles := route.S("handle")
			skip := false
			if handles != nil {
				for _, h := range handles.Children() {
					if handler := h.S("handler"); handler != nil {
						if str, ok := handler.Data().(string); ok && str == "websocket2layer4" {
							skip = true
							break
						}
					}
				}
			}
			if !skip {
				filtered = append(filtered, raw)
			}
		}

		// Replace routes array with filtered slice
		root.Set(filtered, "apps", "http", "servers", srvName, "routes")
	}
	return nil
}

func arrayPrepend(config *gabs.Container, path string, elems ...*gabs.Container) error {
	container := gabs.New()
	container.Array()

	// New elements
	for _, elem := range elems {
		container.ArrayAppend(elem)
	}

	// Old elements
	oldContainer := config.Path(path)
	oldChildren := oldContainer.Children()

	for _, elem := range oldChildren {
		container.ArrayAppend(elem)
	}

	// Save
	config.SetP(container, path)

	return nil
}

func findServerKey(g *gabs.Container, outlineServer *models.OutlineServer) (string, error) {
	config := outlineServer.OutlineConfig()

	// Navigate to apps.http.servers
	servers := g.Path("apps.http.servers")
	if servers.Data() == nil {
		return "", fmt.Errorf("path apps.http.servers not found")
	}

	// Map of server name → *Container
	children := servers.ChildrenMap()

	// Array of priority ports
	ports := []int{config.TCP.Port, config.UDP.Port, 80, 443}

	// 1) Look for any listen entry ending with specific port
	for _, port := range ports {
		if port == 0 {
			continue // Skip zero ports
		}

		for key, srv := range children {
			if listenRaw, ok := srv.S("listen").Data().([]interface{}); ok {
				for _, item := range listenRaw {
					if str, ok := item.(string); ok && (strings.HasSuffix(str, fmt.Sprintf(":%d", port)) || strings.HasSuffix(str, fmt.Sprintf(":%d/tcp", port)) || strings.HasSuffix(str, fmt.Sprintf(":%d/udp", port))) {
						return key, nil
					}
				}
			}
		}
	}

	// 2) Fallback to first key in map
	for key := range children {
		return key, nil
	}

	return "", fmt.Errorf("no servers defined under apps.http.servers")
}
