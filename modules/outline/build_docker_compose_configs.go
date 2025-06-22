package outline

import (
	"fmt"
	"path/filepath"

	"github.com/docker-pet/backend/helpers"
	"github.com/docker-pet/backend/models"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/net/publicsuffix"
	"gopkg.in/yaml.v3"
)

func (m *OutlineModule) buildDockerComposeConfigs() {
	// Get all servers
	servers, err := m.GetAllServers()
	if err != nil {
		m.Logger.Error("Failed to get Outline servers", "Error", err)
		return
	}

	// Fill configs
	for _, server := range servers {
		configPath := filepath.Join(m.Config.OutlineStoragePath, server.Slug(), "docker-compose.yaml")
		configDir := filepath.Dir(configPath)
		serviceName := fmt.Sprintf("outline_%s", server.Slug())

		syncRemoteConfig := server.SyncRemoteConfig()
		outlineConfig := server.OutlineConfig()

		servicePassword := ""
		if server.SyncType() == models.OutlineRemoteSync {
			servicePassword = syncRemoteConfig.RemoteAdminBasicAuth.Password
		}

		// Parse existing config service password if it exists
		servicePasswordHash := ""
		servicePasswordByte := []byte(servicePassword)
		if existsConfigContent, err := helpers.ReadFileIfExists(configPath); err == nil {
			servicePasswordHash = readDockerComposeServicePassword(existsConfigContent)
			if servicePasswordHash != "" && bcrypt.CompareHashAndPassword([]byte(servicePasswordHash), servicePasswordByte) != nil {
				servicePasswordHash = ""
			}
		}

		// Ports
		portsContent := []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "80:80"},
			{Kind: yaml.ScalarNode, Value: "443:443"},
		}

		ports := []int{outlineConfig.TCP.Port, outlineConfig.UDP.Port}
		for _, port := range ports {
			if port == 0 || port == 80 || port == 443 {
				continue // Skip default ports
			}
			portsContent = append(portsContent, &yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: fmt.Sprintf("%d:%d", port, port),
			})
		}

		// Service password
		if servicePasswordHash == "" {
			hash, err := bcrypt.GenerateFromPassword(servicePasswordByte, bcrypt.DefaultCost)
			if err != nil {
				m.Logger.Error(
					"Failed to generate bcrypt hash for Outline service password",
					"Error", err,
					"Server", server.Slug(),
					"ServerID", server.Id,
				)
				continue
			}
			servicePasswordHash = string(hash)
		}

		// Cloudflare API token
		cloudflareApiTokenNode := yaml.Node{Kind: yaml.ScalarNode, Value: "${CLOUDFLARE_API_TOKEN}"}
		if m.Config.CaddyCloudflareApiToken != "" {
			cloudflareApiTokenNode.Value = m.Config.CaddyCloudflareApiToken
			cloudflareApiTokenNode.Style = yaml.DoubleQuotedStyle
		}

		// Cloudflare domain zone
		cloudflareDomainZoneNode := yaml.Node{Kind: yaml.ScalarNode, Value: "${CLOUDFLARE_DOMAIN_ZONE}"}
		cloudflareDomainZone, err := publicsuffix.EffectiveTLDPlusOne(m.formatJobDomain(server))
		if err == nil {
			cloudflareDomainZoneNode.Value = cloudflareDomainZone
		}

		// Environment variables
		environmentContent := []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "CLOUDFLARE_API_TOKEN"},
			&cloudflareApiTokenNode,
			{Kind: yaml.ScalarNode, Value: "CLOUDFLARE_DOMAIN_ZONE"},
			&cloudflareDomainZoneNode,
		}

		// If is remote server, add additional environment variables
		if server.SyncType() == models.OutlineRemoteSync {
			environmentContent = append(environmentContent,
				&yaml.Node{Kind: yaml.ScalarNode, Value: "CADDY_SERVICE_DOMAIN"},
				&yaml.Node{Kind: yaml.ScalarNode, Value: m.formatJobDomain(server)},
				&yaml.Node{Kind: yaml.ScalarNode, Value: "CADDY_SERVICE_PATH"},
				&yaml.Node{Kind: yaml.ScalarNode, Value: "${CADDY_SERVICE_PATH}"},
				&yaml.Node{Kind: yaml.ScalarNode, Value: "CADDY_SERVICE_PASSWORD"},
				&yaml.Node{Kind: yaml.ScalarNode, Value: helpers.EscapeEnv(string(servicePasswordHash)), Style: yaml.DoubleQuotedStyle},
			)
		}

		// volumes
		volumesContent := []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "caddy-data:/data"},
			{Kind: yaml.ScalarNode, Value: "caddy-config:/config"},
		}

		if server.SyncType() == models.OutlineLocalSync {
			volumesContent = []*yaml.Node{
				{Kind: yaml.ScalarNode, Value: "./apps/outline/data/" + server.Slug() + "/data:/data"},
				{Kind: yaml.ScalarNode, Value: "./apps/outline/generated/" + server.Slug() + ":/outline_generated"},
				{Kind: yaml.ScalarNode, Value: "./apps/caddy.prod/data/config:/config"},
			}
		}

		// service
		serviceContent := []*yaml.Node{
			// outline:
			{
				Kind:  yaml.ScalarNode,
				Value: serviceName,
			},
			{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					// image
					{Kind: yaml.ScalarNode, Value: "image"},
					{Kind: yaml.ScalarNode, Value: "ghcr.io/docker-pet/caddy:latest"},
					// container_name
					{Kind: yaml.ScalarNode, Value: "container_name"},
					{Kind: yaml.ScalarNode, Value: serviceName},
					// restart
					{Kind: yaml.ScalarNode, Value: "restart"},
					{Kind: yaml.ScalarNode, Value: "always"},
					// ports
					{Kind: yaml.ScalarNode, Value: "ports"},
					{
						Kind:    yaml.SequenceNode,
						Content: portsContent,
					},
					// environment
					{Kind: yaml.ScalarNode, Value: "environment"},
					{
						Kind:    yaml.MappingNode,
						Content: environmentContent,
					},
					// volumes
					{Kind: yaml.ScalarNode, Value: "volumes"},
					{
						Kind:    yaml.SequenceNode,
						Content: volumesContent,
					},
				},
			},
		}

		// command
		if server.SyncType() == models.OutlineLocalSync {
			serviceContent = append(serviceContent,
				&yaml.Node{Kind: yaml.ScalarNode, Value: "command"},
				&yaml.Node{Kind: yaml.ScalarNode, Value: "run --config /outline_generated/caddy.config.json --adapter json --watch"},
			)
		}

		// Create config
		root := &yaml.Node{
			Kind:        yaml.DocumentNode,
			HeadComment: "This file was generated automatically.\nPlease use it as a template for running Outline VPN Server.",
			Content: []*yaml.Node{
				{
					Kind: yaml.MappingNode,
					Content: []*yaml.Node{
						// services:
						{
							Kind:  yaml.ScalarNode,
							Value: "services",
						},
						{
							Kind:    yaml.MappingNode,
							Content: serviceContent,
						},
					},
				},
			},
		}

		// volumes:
		if server.SyncType() == models.OutlineRemoteSync {
			root.Content[0].Content = append(root.Content[0].Content,
				&yaml.Node{
					Kind:  yaml.ScalarNode,
					Value: "volumes",
				},
				&yaml.Node{
					Kind: yaml.MappingNode,
					Content: []*yaml.Node{
						{Kind: yaml.ScalarNode, Value: "caddy-data"},
						{Kind: yaml.MappingNode},
						{Kind: yaml.ScalarNode, Value: "caddy-config"},
						{Kind: yaml.MappingNode},
					},
				},
			)
		}

		// Save the config
		helpers.EnsureDir(configDir)
		outContent, err := yaml.Marshal(root)
		if err != nil {
			m.Logger.Error("Failed to marshal Outline Docker Compose config", "Error", err)
			continue
		}

		updated, err := helpers.WriteFileIfChanged(
			configPath,
			[]byte(outContent),
			0644,
		)
		if err != nil {
			m.Logger.Error(
				"Failed to write Outline Docker Compose config",
				"Error", err,
				"Server", server.Slug(),
				"ServerID", server.Id,
			)
			continue
		}

		if updated {
			m.Logger.Info(
				"Outline Docker Compose config updated successfully",
				"Server", server.Slug(),
				"ServerID", server.Id,
			)
		}
	}

	// TODO: Remove old folders
}

func readDockerComposeServicePassword(content []byte) string {
	var data interface{}
	if err := yaml.Unmarshal(content, &data); err != nil {
		return ""
	}
	var walk func(interface{}) string
	walk = func(u interface{}) string {
		switch v := u.(type) {
		case map[string]interface{}:
			if pwd, ok := v["CADDY_SERVICE_PASSWORD"].(string); ok {
				return pwd
			}
			for _, val := range v {
				if res := walk(val); res != "" {
					return res
				}
			}
		case []interface{}:
			for _, item := range v {
				if res := walk(item); res != "" {
					return res
				}
			}
		}
		return ""
	}
	return walk(data)
}
