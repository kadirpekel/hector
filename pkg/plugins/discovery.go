package plugins

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

type DiscoveryConfig struct {
	Enabled            bool     `yaml:"enabled" json:"enabled"`
	Paths              []string `yaml:"paths" json:"paths"`
	ScanSubdirectories bool     `yaml:"scan_subdirectories" json:"scan_subdirectories"`
}

type PluginDiscovery struct {
	config *DiscoveryConfig
}

func NewPluginDiscovery(config *DiscoveryConfig) *PluginDiscovery {
	if config == nil {
		config = &DiscoveryConfig{
			Enabled:            true,
			Paths:              []string{"./plugins", "~/.hector/plugins"},
			ScanSubdirectories: true,
		}
	}
	return &PluginDiscovery{
		config: config,
	}
}

type DiscoveredPlugin struct {
	Name         string
	Path         string
	ManifestPath string
	Manifest     *PluginManifest
}

func (d *PluginDiscovery) DiscoverPlugins(ctx context.Context) ([]*DiscoveredPlugin, error) {
	if !d.config.Enabled {
		return nil, nil
	}

	var discovered []*DiscoveredPlugin
	seen := make(map[string]bool)

	for _, path := range d.config.Paths {

		expandedPath := expandPath(path)

		if _, err := os.Stat(expandedPath); os.IsNotExist(err) {
			continue
		}

		plugins, err := d.scanPath(ctx, expandedPath)
		if err != nil {
			return nil, fmt.Errorf("failed to scan path '%s': %w", path, err)
		}

		for _, plugin := range plugins {
			if !seen[plugin.Path] {
				discovered = append(discovered, plugin)
				seen[plugin.Path] = true
			}
		}
	}

	return discovered, nil
}

func (d *PluginDiscovery) scanPath(ctx context.Context, path string) ([]*DiscoveredPlugin, error) {
	var plugins []*DiscoveredPlugin

	manifestPath := path + ".plugin.yaml"
	if _, err := os.Stat(manifestPath); err == nil {
		plugin, err := d.loadPluginFromManifest(path, manifestPath)
		if err != nil {
			return nil, err
		}
		if plugin != nil {
			plugins = append(plugins, plugin)
		}
		return plugins, nil
	}

	if d.config.ScanSubdirectories {
		err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			if strings.HasSuffix(filePath, ".plugin.yaml") {

				execPath := strings.TrimSuffix(filePath, ".plugin.yaml")
				plugin, err := d.loadPluginFromManifest(execPath, filePath)
				if err != nil {

					fmt.Printf("Warning: Failed to load plugin manifest '%s': %v\n", filePath, err)
					return nil
				}
				if plugin != nil {
					plugins = append(plugins, plugin)
				}
			}

			return nil
		})

		if err != nil {
			return nil, fmt.Errorf("failed to walk directory '%s': %w", path, err)
		}
	} else {

		entries, err := os.ReadDir(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read directory '%s': %w", path, err)
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			name := entry.Name()
			if strings.HasSuffix(name, ".plugin.yaml") {
				filePath := filepath.Join(path, name)
				execPath := strings.TrimSuffix(filePath, ".plugin.yaml")
				plugin, err := d.loadPluginFromManifest(execPath, filePath)
				if err != nil {
					fmt.Printf("Warning: Failed to load plugin manifest '%s': %v\n", filePath, err)
					continue
				}
				if plugin != nil {
					plugins = append(plugins, plugin)
				}
			}
		}
	}

	return plugins, nil
}

func (d *PluginDiscovery) loadPluginFromManifest(execPath, manifestPath string) (*DiscoveredPlugin, error) {

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifestWrapper struct {
		Plugin PluginManifest `yaml:"plugin"`
	}
	if err := yaml.Unmarshal(data, &manifestWrapper); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	manifest := &manifestWrapper.Plugin

	if err := d.validateManifest(manifest); err != nil {
		return nil, fmt.Errorf("invalid manifest: %w", err)
	}

	if _, err := os.Stat(execPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("plugin executable not found: %s", execPath)
	}

	info, err := os.Stat(execPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat executable: %w", err)
	}

	if info.Mode()&0111 == 0 {
		return nil, fmt.Errorf("plugin is not executable: %s", execPath)
	}

	return &DiscoveredPlugin{
		Name:         manifest.Name,
		Path:         execPath,
		ManifestPath: manifestPath,
		Manifest:     manifest,
	}, nil
}

func (d *PluginDiscovery) validateManifest(manifest *PluginManifest) error {
	if manifest.Name == "" {
		return fmt.Errorf("manifest missing 'name' field")
	}
	if manifest.Version == "" {
		return fmt.Errorf("manifest missing 'version' field")
	}
	if manifest.Type == "" {
		return fmt.Errorf("manifest missing 'type' field")
	}
	if manifest.Protocol == "" {
		return fmt.Errorf("manifest missing 'protocol' field")
	}

	validTypes := map[PluginType]bool{
		PluginTypeLLM:            true,
		PluginTypeDatabase:       true,
		PluginTypeEmbedder:       true,
		PluginTypeTool:           true,
		PluginTypeReasoning:      true,
		PluginTypeDocumentParser: true,
	}
	if !validTypes[manifest.Type] {
		return fmt.Errorf("invalid plugin type: %s", manifest.Type)
	}

	if manifest.Protocol != ProtocolGRPC {
		return fmt.Errorf("invalid protocol: %s (only 'grpc' is supported)", manifest.Protocol)
	}

	return nil
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

func FilterByType(plugins []*DiscoveredPlugin, pluginType PluginType) []*DiscoveredPlugin {
	var filtered []*DiscoveredPlugin
	for _, plugin := range plugins {
		if plugin.Manifest != nil && plugin.Manifest.Type == pluginType {
			filtered = append(filtered, plugin)
		}
	}
	return filtered
}

func FilterByProtocol(plugins []*DiscoveredPlugin, protocol PluginProtocol) []*DiscoveredPlugin {
	var filtered []*DiscoveredPlugin
	for _, plugin := range plugins {
		if plugin.Manifest != nil && plugin.Manifest.Protocol == protocol {
			filtered = append(filtered, plugin)
		}
	}
	return filtered
}

func FilterByName(plugins []*DiscoveredPlugin, name string) *DiscoveredPlugin {
	for _, plugin := range plugins {
		if plugin.Manifest != nil && plugin.Manifest.Name == name {
			return plugin
		}
	}
	return nil
}
