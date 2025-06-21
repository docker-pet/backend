package outline

import (
	"fmt"
	"path"
	"path/filepath"

	"github.com/docker-pet/backend/helpers"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/net/publicsuffix"
	"gopkg.in/yaml.v3"
)

func (m *OutlineModule) buildDockerComposeConfigs() {
	// Create folders
	err := helpers.EnsureDir(m.Config.OutlineStoragePath)
	if err != nil {
		m.Logger.Error(
			"Failed to create Outline configs directory",
			"Error", err,
			"Path", m.Config.OutlineStoragePath,
		)
		return
	}

	// Get all servers
	servers, err := m.GetAllServers()
	if err != nil {
		m.Logger.Error("Failed to get Outline servers", "Error", err)
		return
	}

	// List all existing target files
	files, err := helpers.MapFileNamesToPaths(m.Config.OutlineStoragePath, "yaml")
	if err != nil {
		m.Logger.Error(
			"Failed to read Prometheus targets directory",
			"Error", err,
		)
		return
	}

	// Fill configs
	for _, server := range servers {
		configFileName := fmt.Sprintf("docker-compose.%s.yaml", server.Slug())
		configPath := filepath.Join(m.Config.OutlineStoragePath, configFileName)
		serviceName := fmt.Sprintf("outline_%s", server.Slug())
		delete(files, configFileName)

		// Parse existing config service password if it exists
		servicePasswordHash := ""
		servicePasswordByte := []byte(server.ServicePassword())
		if existsConfigContent, err := helpers.ReadFileIfExists(configPath); err == nil {
			servicePasswordHash = readDockerComposeServicePassword(existsConfigContent)
			if servicePasswordHash != "" && bcrypt.CompareHashAndPassword([]byte(servicePasswordHash), servicePasswordByte) != nil {
				servicePasswordHash = ""
			}
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
							Kind: yaml.MappingNode,
							Content: []*yaml.Node{
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
											Kind: yaml.SequenceNode,
											Content: []*yaml.Node{
												{Kind: yaml.ScalarNode, Value: "80:80"},
												{Kind: yaml.ScalarNode, Value: "443:443"},
											},
										},
										// environment
										{Kind: yaml.ScalarNode, Value: "environment"},
										{
											Kind: yaml.MappingNode,
											Content: []*yaml.Node{
												{Kind: yaml.ScalarNode, Value: "CLOUDFLARE_API_TOKEN"},
												&cloudflareApiTokenNode,
												{Kind: yaml.ScalarNode, Value: "CLOUDFLARE_DOMAIN_ZONE"},
												&cloudflareDomainZoneNode,
												{Kind: yaml.ScalarNode, Value: "CADDY_SERVICE_DOMAIN"},
												{Kind: yaml.ScalarNode, Value: m.formatJobDomain(server)},
												{Kind: yaml.ScalarNode, Value: "CADDY_SERVICE_PATH"},
												{Kind: yaml.ScalarNode, Value: path.Join("/", server.ServicePath())},
												{Kind: yaml.ScalarNode, Value: "CADDY_SERVICE_PASSWORD"},
												{Kind: yaml.ScalarNode, Value: helpers.EscapeEnv(string(servicePasswordHash)), Style: yaml.DoubleQuotedStyle},
											},
										},
										// volumes
										{Kind: yaml.ScalarNode, Value: "volumes"},
										{
											Kind: yaml.SequenceNode,
											Content: []*yaml.Node{
												{Kind: yaml.ScalarNode, Value: "caddy-data:/data"},
												{Kind: yaml.ScalarNode, Value: "caddy-config:/config"},
											},
										},
									},
								},
							},
						},
						// volumes:
						{
							Kind:  yaml.ScalarNode,
							Value: "volumes",
						},
						{
							Kind: yaml.MappingNode,
							Content: []*yaml.Node{
								{Kind: yaml.ScalarNode, Value: "caddy-data"},
								{Kind: yaml.MappingNode},
								{Kind: yaml.ScalarNode, Value: "caddy-config"},
								{Kind: yaml.MappingNode},
							},
						},
					},
				},
			},
		}

		// Save the config
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

	// Remove old files
	for fileName, filePath := range files {
		err := helpers.RemoveFileIfExists(filePath)
		if err != nil {
			m.Logger.Error(
				"Failed to remove old Outline Docker Compose config file",
				"FileName", fileName,
				"Error", err,
			)
			continue
		}

		m.Logger.Info(
			"Removed old Outline Docker Compose config file",
			"FileName", fileName,
		)
	}
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
