package outline

import (
	"fmt"
	"path"
	"strings"

	"github.com/Jeffail/gabs/v2"
	"github.com/docker-pet/backend/models"
)

func (m *OutlineModule) configureCaddy(serverId string) {
	tokens := m.tokenStore.GetAllByServer(serverId)
	server, err := m.GetServerById(serverId)
	if err != nil {
		m.Logger.Warn(
			"Failed to get server by ID",
			"Error", err,
			"ServerId", serverId,
		)
		return
	}

	caddyAdminConfigEndpoint := fmt.Sprintf("https://%s/%s/", m.formatJobDomain(server), path.Join(server.ServicePath(), "config"))

	// Fetch the Caddy configuration
	configResponse, err := m.Ctx.HttpClient.R().
		SetBasicAuth(m.Config.CaddyBasicAuthUsername, server.ServicePassword()).
		Get(caddyAdminConfigEndpoint)

	// log response as text
	if err != nil {
		m.Logger.Warn(
			"Failed to fetch Caddy configuration",
			"Error", err,
			"ServerId", server.Id,
		)
		return
	}

	if configResponse.StatusCode() != 200 {
		m.Logger.Warn(
			"Failed to fetch Caddy configuration",
			"Status", configResponse.Status(),
			"ServerId", server.Id,
			"Response", configResponse.String(),
		)
		return
	}

	// Parse the Caddy configuration
	config, err := gabs.ParseJSON(configResponse.Bytes())
	if err != nil {
		m.Logger.Warn(
			"Failed to parse Caddy configuration",
			"Error", err,
			"ServerId", server.Id,
			"Response", configResponse.String(),
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
	serverKey, err := findServerKey(config)
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

	// Save config
	_, err = m.Ctx.HttpClient.R().
		SetBasicAuth(m.Config.CaddyBasicAuthUsername, server.ServicePassword()).
		SetContentType("application/json").
		SetBody(config.String()).
		Patch(caddyAdminConfigEndpoint)
	if err != nil {
		m.Logger.Warn(
			"Failed to save Caddy configuration",
			"Error", err,
			"ServerId", server.Id,
			"ServerSlug", server.Slug(),
		)
	}
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

	handleType := "stream"
	serverPath := server.TCPPath()
	if packageType == "udp" {
		handleType = "packet"
		serverPath = server.UDPPath()
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

	matchHost := gabs.New()
	matchHost.Array()
	matchHost.ArrayAppend(m.formatJobDomain(server))
	match.SetP(matchHost, "host")

	matchPath := gabs.New()
	matchPath.Array()
	matchPath.ArrayAppend(path.Join("/", serverPath))
	match.SetP(matchPath, "path")

	return route
}

func (m *OutlineModule) hasKeysChanges(config *gabs.Container, tokens []*Token) bool {
	// Собираем актуальные данные по токенам
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

func findServerKey(g *gabs.Container) (string, error) {
	// Navigate to apps.http.servers
	servers := g.Path("apps.http.servers")
	if servers.Data() == nil {
		return "", fmt.Errorf("path apps.http.servers not found")
	}

	// Map of server name → *Container
	children := servers.ChildrenMap()

	// 1) Look for any listen entry ending with ":443"
	for key, srv := range children {
		if listenRaw, ok := srv.S("listen").Data().([]interface{}); ok {
			for _, item := range listenRaw {
				if str, ok := item.(string); ok && strings.HasSuffix(str, ":443") {
					return key, nil
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
