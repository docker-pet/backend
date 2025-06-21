package outline

import (
	"fmt"
	"path/filepath"

	"github.com/docker-pet/backend/helpers"
	"gopkg.in/yaml.v3"
)

func (m *OutlineModule) buildPrometheusConfig() bool {
	prometheusConfigPath := filepath.Join(m.Config.PrometheusStoragePath, "prometheus.yml")
	prometheusTargetsPath := filepath.Join(m.Config.PrometheusStoragePath, m.getPrometheusTargetsRelativePath())

	prometheusConfigContent := GetDefaultPrometheusConfig()

	if existsConfigContent, err := helpers.ReadFileIfExists(prometheusConfigPath); err == nil {
		prometheusConfigContent = existsConfigContent
	}

	var root yaml.Node
	if err := yaml.Unmarshal(prometheusConfigContent, &root); err != nil {
		panic(err)
	}

	// Find or create the mapping node
	var mapping *yaml.Node
	if len(root.Content) > 0 && root.Content[0].Kind == yaml.MappingNode {
		mapping = root.Content[0]
	} else {
		mapping = &yaml.Node{Kind: yaml.MappingNode}
		if len(root.Content) == 0 {
			root.Content = append(root.Content, mapping)
		} else {
			root.Content[0] = mapping
		}
	}

	// Create the required job block
	jobNode := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{
				Kind:  yaml.ScalarNode,
				Value: "job_name",
			},
			{
				Kind:        yaml.ScalarNode,
				Value:       m.Config.PrometheusJobName,
				LineComment: "This job is auto-managed; manual edits will be overwritten.",
			},
			{
				Kind:  yaml.ScalarNode,
				Value: "scheme",
			},
			{
				Kind:  yaml.ScalarNode,
				Value: "https",
			},
			{
				Kind:  yaml.ScalarNode,
				Value: "file_sd_configs",
			},
			{
				Kind: yaml.SequenceNode,
				Content: []*yaml.Node{
					{
						Kind: yaml.MappingNode,
						Content: []*yaml.Node{
							{
								Kind:  yaml.ScalarNode,
								Value: "files",
							},
							{
								Kind: yaml.SequenceNode,
								Content: []*yaml.Node{
									{Kind: yaml.ScalarNode, Value: m.getPrometheusTargetsRelativePath() + "*.yml"},
								},
							},
						},
					},
				},
			},
		},
	}

	// Replace or add the block in scrape_configs
	var scrapeConfigsNode *yaml.Node
	for i := 0; i < len(mapping.Content); i += 2 {
		key, val := mapping.Content[i], mapping.Content[i+1]
		if key.Value == "scrape_configs" && val.Kind == yaml.SequenceNode {
			scrapeConfigsNode = val
			break
		}
	}
	if scrapeConfigsNode == nil {
		scrapeConfigsNode = &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		mapping.Content = append(mapping.Content, &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: "scrape_configs",
		}, scrapeConfigsNode)
	}

	// Search and replace or add the job block
	replaced := false
	for idx, sc := range scrapeConfigsNode.Content {
		if sc.Kind == yaml.MappingNode {
			for i := 0; i < len(sc.Content); i += 2 {
				k, v := sc.Content[i], sc.Content[i+1]
				if k.Value == "job_name" && v.Value == m.Config.PrometheusJobName {
					scrapeConfigsNode.Content[idx] = jobNode
					replaced = true
					break
				}
			}
		}
		if replaced {
			break
		}
	}
	if !replaced {
		scrapeConfigsNode.Content = append(scrapeConfigsNode.Content, jobNode)
	}

	// Create folders
	err := helpers.EnsureDir(prometheusTargetsPath)
	if err != nil {
		m.Logger.Error("Failed to create Prometheus targets directory", "Error", err)
		return false
	}

	// Save the config
	outContent, err := yaml.Marshal(&root)
	if err != nil {
		m.Logger.Error("Failed to marshal Prometheus config", "Error", err)
		return false
	}
	updated, err := helpers.WriteFileIfChanged(
		prometheusConfigPath,
		[]byte(outContent),
		0644,
	)
	if err != nil {
		m.Logger.Error("Failed to write Prometheus config", "Error", err)
		return false
	}
	if updated {
		m.Logger.Info("Prometheus config updated successfully, need to restart Prometheus")
	}
	return updated
}

func (m *OutlineModule) buildPrometheusTargets() {
	prometheusTargetsPath := filepath.Join(m.Config.PrometheusStoragePath, m.getPrometheusTargetsRelativePath())
	servers, err := m.GetAllServers()
	if err != nil {
		m.Logger.Error("Failed to get Outline servers", "Error", err)
		return
	}

	// List all existing target files
	files, err := helpers.MapFileNamesToPaths(prometheusTargetsPath, "yml")
	if err != nil {
		m.Logger.Error(
			"Failed to read Prometheus targets directory",
			"Error", err,
		)
		return
	}

	// Fill configs
	for _, server := range servers {
		if !server.Enabled() {
			continue // Skip disabled servers
		}

		// Config file name
		configFileName := fmt.Sprintf("%s.yml", server.Slug())
		delete(files, configFileName)

		// Create config
		targetsNode := &yaml.Node{
			Kind:        yaml.SequenceNode,
			HeadComment: "This file is auto-managed; manual edits will be overwritten.",
			Content: []*yaml.Node{
				{
					Kind: yaml.MappingNode,
					Content: []*yaml.Node{
						{
							Kind:  yaml.ScalarNode,
							Value: "targets",
						},
						{
							Kind: yaml.SequenceNode,
							Content: []*yaml.Node{
								{Kind: yaml.ScalarNode, Value: m.appConfig.AppConfig().AppDomain()},
							},
						},
						{
							Kind:  yaml.ScalarNode,
							Value: "labels",
						},
						{
							Kind: yaml.MappingNode,
							Content: []*yaml.Node{
								{Kind: yaml.ScalarNode, Value: "__metrics_path__"},
								{Kind: yaml.ScalarNode, Value: fmt.Sprintf("/api/outline/metics/%s/%s", server.Id, server.MetricsSecret())},
								{Kind: yaml.ScalarNode, Value: "server_slug"},
								{Kind: yaml.ScalarNode, Value: server.Slug()},
								{Kind: yaml.ScalarNode, Value: "server_id"},
								{Kind: yaml.ScalarNode, Value: server.Id},
							},
						},
					},
				},
			},
		}

		// Save the config
		outContent, err := yaml.Marshal(targetsNode)
		if err != nil {
			m.Logger.Error("Failed to marshal Prometheus targets", "Error", err)
			continue
		}

		updated, err := helpers.WriteFileIfChanged(
			filepath.Join(prometheusTargetsPath, configFileName),
			[]byte(outContent),
			0644,
		)
		if err != nil {
			m.Logger.Error(
				"Failed to write Prometheus target config",
				"Error", err,
				"Server", server.Slug(),
				"ServerID", server.Id,
			)
			continue
		}

		if updated {
			m.Logger.Info(
				"Prometheus target config updated successfully",
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
				"Failed to remove old Prometheus target file",
				"FileName", fileName,
				"Error", err,
			)
			continue
		}

		m.Logger.Info(
			"Removed old Prometheus target file",
			"FileName", fileName,
		)
	}
}

func (m *OutlineModule) getPrometheusTargetsRelativePath() string {
	return fmt.Sprintf("./%s_targets/", m.Config.PrometheusJobName)
}

func GetDefaultPrometheusConfig() []byte {
	return []byte(`
global:
  scrape_interval: 15s
  scrape_timeout: 10s
  evaluation_interval: 15s

storage:
  retention: 64d
`)
}
