package outline

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/docker-pet/backend/models"
	"github.com/pocketbase/pocketbase/core"
	"gopkg.in/yaml.v3"
)

func (m *OutlineModule) registerOutlineConnectEndpoint() {
	sendError := func(e *core.RequestEvent, message string, details string) error {
		m.Logger.Error(message, "Details", details)
		e.Response.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		return e.Blob(http.StatusOK, "application/x-yaml", []byte("error:\n  message: "+message+"\n  details: "+details))
	}

	m.Ctx.App.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/api/outline/{userId}/{outlineSecret}", func(e *core.RequestEvent) error {
			userId := e.Request.PathValue("userId")
			outlineSecret := e.Request.PathValue("outlineSecret")

			// Validate user
			user, err := m.users.GetUserById(userId)
			if err != nil {
				return sendError(
					e,
					"User not found",
					"The user with the specified ID does not exist.",
				)
			}

			// Validate Outline Secret
			if user.OutlineToken() != outlineSecret {
				return sendError(
					e,
					"Invalid Outline Secret",
					"The provided Outline secret is invalid or does not match the user's secret.",
				)
			}

			// Guest
			if !user.IsActive() {
				return sendError(
					e,
					"Guest Access Denied",
					"Guests are not allowed to connect to Outline.",
				)
			}

			servers, err := m.GetAllActiveServers()
			if err != nil {
				return sendError(
					e,
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
				return sendError(
					e,
					"No Active Outline Servers",
					"There are currently no active Outline servers available for connection.",
				)
			}

			var server *models.OutlineServer

			// Has picked server by user
			if user.OutlineServer() != "" {
				for _, s := range availableServers {
					if s.Id == user.OutlineServer() {
						server = s
						break
					}
				}
			}

			// Random server
			if server == nil {
				rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
				server = availableServers[rnd.Intn(len(availableServers))]
			}

			// Token
			token, err := m.tokenStore.GetOrGenerate(user.Id, server.Id)
			if err != nil {
				return sendError(
					e,
					"Failed to generate Outline token",
					"An error occurred while trying to generate an Outline token for the user.",
				)
			}

			// Ports
			tcpPort := ""
			udpPort := ""
			outlineConfig := server.OutlineConfig()

			if outlineConfig.TCP.Port != 443 && outlineConfig.TCP.Port != 0 {
				tcpPort = fmt.Sprintf(":%d", outlineConfig.TCP.Port)
			}
			if outlineConfig.UDP.Port != 443 && outlineConfig.UDP.Port != 0 {
				udpPort = fmt.Sprintf(":%d", outlineConfig.UDP.Port)
			}

			// Tcp Settings
			tcpContent := []*yaml.Node{
				{Kind: yaml.ScalarNode, Value: "$type"},
				{Kind: yaml.ScalarNode, Value: "shadowsocks"},

				{Kind: yaml.ScalarNode, Value: "endpoint"},
				{
					Kind: yaml.MappingNode,
					Content: []*yaml.Node{
						{Kind: yaml.ScalarNode, Value: "$type"},
						{Kind: yaml.ScalarNode, Value: "websocket"},
						{Kind: yaml.ScalarNode, Value: "url"},
						{Kind: yaml.ScalarNode, Value: fmt.Sprintf("wss://%s%s/%s", m.formatJobDomain(server), tcpPort, outlineConfig.TCP.Path)},
					},
				},
				{Kind: yaml.ScalarNode, Value: "cipher"},
				{Kind: yaml.ScalarNode, Value: m.Config.OutlineCipher},
				{Kind: yaml.ScalarNode, Value: "secret"},
				{Kind: yaml.ScalarNode, Value: token.Token},
			}

			if outlineConfig.TCP.Prefix != "" && user.OutlinePrefixEnabled() {
				tcpContent = append(makeConnectionPrefix(outlineConfig.TCP), tcpContent...)
			}

			// Udp Settings
			udpContent := []*yaml.Node{
				{Kind: yaml.ScalarNode, Value: "$type"},
				{Kind: yaml.ScalarNode, Value: "shadowsocks"},

				{Kind: yaml.ScalarNode, Value: "endpoint"},
				{
					Kind: yaml.MappingNode,
					Content: []*yaml.Node{
						{Kind: yaml.ScalarNode, Value: "$type"},
						{Kind: yaml.ScalarNode, Value: "websocket"},
						{Kind: yaml.ScalarNode, Value: "url"},
						{Kind: yaml.ScalarNode, Value: fmt.Sprintf("wss://%s%s/%s", m.formatJobDomain(server), udpPort, outlineConfig.UDP.Path)},
					},
				},
				{Kind: yaml.ScalarNode, Value: "cipher"},
				{Kind: yaml.ScalarNode, Value: m.Config.OutlineCipher},
				{Kind: yaml.ScalarNode, Value: "secret"},
				{Kind: yaml.ScalarNode, Value: token.Token},
			}

			if outlineConfig.UDP.Prefix != "" && user.OutlinePrefixEnabled() {
				udpContent = append(makeConnectionPrefix(outlineConfig.UDP), tcpContent...)
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
								Kind:    yaml.MappingNode,
								Content: tcpContent,
							},

							{Kind: yaml.ScalarNode, Value: "udp"},
							{
								Kind:    yaml.MappingNode,
								Content: udpContent,
							},
						},
					},
				},
			}

			// Marshal YAML
			content, err := yaml.Marshal(root)
			if err != nil {
				return sendError(
					e,
					"Config Generation Error",
					"An error occurred while generating the Outline configuration.",
				)
			}

			// Fix prefix escaping
			content = []byte(fixPrefixEscaping(string(content)))

			// Response
			e.Response.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			return e.Blob(http.StatusOK, "application/x-yaml", content)
		})

		se.Router.GET("/api/outline/redirect/{userId}/{outlineSecret}", func(e *core.RequestEvent) error {
			url := fmt.Sprintf(
				"ssconf://%s/api/outline/%s/%s#%s",
				m.appConfig.AppConfig().AppDomain(),
				e.Request.PathValue("userId"),
				e.Request.PathValue("outlineSecret"),
				url.PathEscape(m.appConfig.AppConfig().AppTitle()),
			)
			return e.Redirect(http.StatusMovedPermanently, url)
		})

		return se.Next()
	})
}

func makeConnectionPrefix(protocol *models.OutlineConfigurationProtocol) []*yaml.Node {
	decoded, _ := url.QueryUnescape(protocol.Prefix)
	prefix := ""
	for _, b := range []byte(decoded) {
		prefix += fmt.Sprintf("\\u%04X", b)
	}

	return []*yaml.Node{
		{Kind: yaml.ScalarNode, Value: "prefix"},
		{Kind: yaml.ScalarNode, Value: prefix, Style: yaml.DoubleQuotedStyle},
	}
}

func fixPrefixEscaping(input string) string {
	lines := strings.Split(input, "\n")
	prefixRegex := regexp.MustCompile(`^(\s*prefix:\s*")((?:\\\\u[0-9a-fA-F]{4})+)(")`)

	for i, line := range lines {
		if matches := prefixRegex.FindStringSubmatch(line); matches != nil {
			fixed := regexp.MustCompile(`\\\\u([0-9a-fA-F]{4})`).ReplaceAllString(matches[2], `\u$1`)
			lines[i] = matches[1] + fixed + matches[3]
		}
	}

	return strings.Join(lines, "\n")
}
