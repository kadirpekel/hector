package config

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/hashicorp/consul/api"
	"github.com/mitchellh/mapstructure"
	clientv3 "go.etcd.io/etcd/client/v3"
	"gopkg.in/yaml.v3"
)

type ConfigType string

const (
	ConfigTypeFile      ConfigType = "file"
	ConfigTypeConsul    ConfigType = "consul"
	ConfigTypeEtcd      ConfigType = "etcd"
	ConfigTypeZookeeper ConfigType = "zookeeper"
)

type LoaderOptions struct {
	Type ConfigType

	Path string

	Endpoints []string

	Watch bool

	OnChange func(*Config) error
}

type Loader struct {
	options  LoaderOptions
	stopChan chan struct{}

	// Provider-specific clients
	consulClient *api.Client
	etcdClient   *clientv3.Client
	zkProvider   *ZookeeperProvider
	fileWatcher  *fsnotify.Watcher
}

func NewLoader(opts LoaderOptions) (*Loader, error) {
	if opts.Type == "" {
		opts.Type = ConfigTypeFile
	}

	if opts.Path == "" {
		return nil, fmt.Errorf("config path is required")
	}

	if len(opts.Endpoints) == 0 {
		switch opts.Type {
		case ConfigTypeConsul:
			opts.Endpoints = []string{"localhost:8500"}
		case ConfigTypeEtcd:
			opts.Endpoints = []string{"localhost:2379"}
		case ConfigTypeZookeeper:
			opts.Endpoints = []string{"localhost:2181"}
		}
	}

	loader := &Loader{
		options:  opts,
		stopChan: make(chan struct{}),
	}

	// Initialize provider clients if needed
	switch opts.Type {
	case ConfigTypeConsul:
		consulConfig := api.DefaultConfig()
		consulConfig.Address = opts.Endpoints[0]
		client, err := api.NewClient(consulConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create consul client: %w", err)
		}
		loader.consulClient = client

	case ConfigTypeEtcd:
		etcdConfig := clientv3.Config{
			Endpoints:   opts.Endpoints,
			DialTimeout: 5 * time.Second,
		}
		client, err := clientv3.New(etcdConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create etcd client: %w", err)
		}
		loader.etcdClient = client

	case ConfigTypeZookeeper:
		zkProvider, err := NewZookeeperProvider(opts.Endpoints, opts.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to create zookeeper provider: %w", err)
		}
		loader.zkProvider = zkProvider
	}

	return loader, nil
}

// readBytes reads raw config bytes from the configured source
func (l *Loader) readBytes() ([]byte, error) {
	switch l.options.Type {
	case ConfigTypeFile:
		return os.ReadFile(l.options.Path)

	case ConfigTypeConsul:
		kv, _, err := l.consulClient.KV().Get(l.options.Path, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to read consul key %s: %w", l.options.Path, err)
		}
		if kv == nil {
			return nil, fmt.Errorf("consul key %s not found", l.options.Path)
		}
		return kv.Value, nil

	case ConfigTypeEtcd:
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		resp, err := l.etcdClient.Get(ctx, l.options.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to read etcd key %s: %w", l.options.Path, err)
		}
		if len(resp.Kvs) == 0 {
			return nil, fmt.Errorf("etcd key %s not found", l.options.Path)
		}
		return resp.Kvs[0].Value, nil

	case ConfigTypeZookeeper:
		return l.zkProvider.ReadBytes()

	default:
		return nil, fmt.Errorf("unsupported config type: %s", l.options.Type)
	}
}

// parseBytes parses raw bytes into map[string]interface{}
// Supports both YAML (standard) and JSON (fallback) for all providers
func (l *Loader) parseBytes(data []byte) (map[string]interface{}, error) {
	var result map[string]interface{}

	// Try YAML first (standard format for all providers)
	// YAML is a superset of JSON, so it can parse JSON as well
	if err := yaml.Unmarshal(data, &result); err == nil {
		return result, nil
	}

	// If YAML parsing fails, try JSON as fallback
	// This allows users to use JSON if they prefer, but YAML is the standard
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse config (tried YAML and JSON): %w", err)
	}

	return result, nil
}

// loadAndProcess loads config, expands env vars, validates, and unmarshals
func (l *Loader) loadAndProcess() (*Config, error) {
	// 1. Read raw bytes
	rawBytes, err := l.readBytes()
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	// 2. Parse into map
	rawMap, err := l.parseBytes(rawBytes)
	if err != nil {
		return nil, err
	}

	// 3. Expand environment variables
	expandedData := ExpandEnvVarsInData(rawMap)
	expandedMap, ok := expandedData.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("env var expansion returned unexpected type %T", expandedData)
	}

	// 4. Validate structure
	strictResult, err := ValidateConfigStructure(expandedMap)
	if err != nil {
		return nil, fmt.Errorf("strict validation check failed: %w", err)
	}
	if !strictResult.Valid() {
		return nil, fmt.Errorf("configuration has structural errors:\n%s", strictResult.FormatErrors())
	}

	// 5. Unmarshal into Config struct
	cfg := &Config{}
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:  cfg,
		TagName: "yaml",
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
		),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create decoder: %w", err)
	}
	if err := decoder.Decode(expandedMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// 6. Process pipeline (expand shortcuts, defaults, validate)
	processedCfg, err := ProcessConfigPipeline(cfg)
	if err != nil {
		return nil, fmt.Errorf("config processing failed: %w", err)
	}

	return processedCfg, nil
}

func (l *Loader) Load() (*Config, error) {
	cfg, err := l.loadAndProcess()
	if err != nil {
		return nil, err
	}

	if l.options.Watch {
		go l.watch()
	}

	return cfg, nil
}

func (l *Loader) watch() {
	slog.Info("Config watcher started", "type", l.options.Type)

	switch l.options.Type {
	case ConfigTypeFile:
		l.watchFile()
	case ConfigTypeConsul:
		l.watchConsul()
	case ConfigTypeEtcd:
		l.watchEtcd()
	case ConfigTypeZookeeper:
		l.watchZookeeper()
	}
}

func (l *Loader) watchFile() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		slog.Error("Failed to create file watcher", "error", err)
		return
	}
	defer watcher.Close()
	l.fileWatcher = watcher

	// Watch the directory containing the file, not the file itself
	// (some systems don't support watching files directly)
	configDir := filepath.Dir(l.options.Path)
	configFile := filepath.Base(l.options.Path)

	if err := watcher.Add(configDir); err != nil {
		slog.Error("Failed to watch config directory", "dir", configDir, "error", err)
		return
	}

	slog.Info("Watching config file", "path", l.options.Path)

	var debounceTimer *time.Timer
	const debounceDelay = 100 * time.Millisecond

	for {
		select {
		case <-l.stopChan:
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			return
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			// Only reload if the watched file changed
			if filepath.Base(event.Name) == configFile {
				if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
					// Cancel previous timer if exists
					if debounceTimer != nil {
						debounceTimer.Stop()
					}
					// Start new debounce timer
					debounceTimer = time.AfterFunc(debounceDelay, l.reload)
				} else if event.Op&fsnotify.Remove == fsnotify.Remove {
					slog.Warn("Config file was deleted", "path", l.options.Path)
					// Try to re-add watch if file is recreated
					time.Sleep(500 * time.Millisecond)
					if _, err := os.Stat(l.options.Path); err == nil {
						watcher.Add(configDir)
					}
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			slog.Error("File watcher error", "error", err)
		}
	}
}

func (l *Loader) watchConsul() {
	// Use Consul blocking query
	var lastIndex uint64

	for {
		if l.shouldStopWatching() {
			return
		}

		kv, meta, err := l.consulClient.KV().Get(l.options.Path, &api.QueryOptions{
			WaitIndex: lastIndex,
			WaitTime:  10 * time.Second,
		})

		if err != nil {
			l.handleWatchError(err)
			continue
		}

		if kv == nil {
			slog.Warn("Consul key was deleted", "path", l.options.Path)
			return
		}

		// Only reload if index actually changed (actual update, not just timeout)
		if meta != nil && meta.LastIndex != lastIndex && lastIndex > 0 {
			l.reload()
		}

		if meta != nil {
			lastIndex = meta.LastIndex
		}
	}
}

func (l *Loader) watchEtcd() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		select {
		case <-l.stopChan:
			cancel()
		case <-ctx.Done():
		}
	}()

	rch := l.etcdClient.Watch(ctx, l.options.Path)
	for {
		select {
		case <-l.stopChan:
			return
		case wresp, ok := <-rch:
			if !ok {
				return
			}
			if wresp.Canceled {
				slog.Warn("Etcd watch canceled", "error", wresp.Err())
				return
			}
			if wresp.Err() != nil {
				l.handleWatchError(wresp.Err())
				continue
			}
			for _, ev := range wresp.Events {
				if ev.Type == clientv3.EventTypePut {
					l.reload()
				} else if ev.Type == clientv3.EventTypeDelete {
					slog.Warn("Etcd key was deleted", "path", l.options.Path)
					return
				}
			}
		}
	}
}

func (l *Loader) watchZookeeper() {
	if l.zkProvider == nil {
		return
	}

	l.zkProvider.Watch(func(event interface{}, err error) {
		if l.shouldStopWatching() {
			return
		}

		if err != nil {
			l.handleWatchError(err)
			return
		}

		l.reload()
	})
}

// shouldStopWatching checks if the loader should stop watching
func (l *Loader) shouldStopWatching() bool {
	select {
	case <-l.stopChan:
		return true
	default:
		return false
	}
}

// handleWatchError handles watch errors with consistent logging and backoff
func (l *Loader) handleWatchError(err error) {
	slog.Warn("Watch error", "error", err)
	time.Sleep(1 * time.Second)
}

func (l *Loader) reload() {
	newCfg, err := l.loadAndProcess()
	if err != nil {
		slog.Error("Failed to reload config", "error", err)
		return
	}

	if l.options.OnChange != nil {
		if err := l.options.OnChange(newCfg); err != nil {
			slog.Error("Config change callback failed", "error", err)
		} else {
			slog.Info("Configuration reloaded successfully", "type", l.options.Type)
		}
	}
}

func (l *Loader) Stop() {
	// Use select to avoid closing closed channel
	select {
	case <-l.stopChan:
		// Already closed
	default:
		close(l.stopChan)
	}

	if l.fileWatcher != nil {
		l.fileWatcher.Close()
		l.fileWatcher = nil
	}
	if l.etcdClient != nil {
		l.etcdClient.Close()
		l.etcdClient = nil
	}
	if l.zkProvider != nil {
		l.zkProvider.Close()
		l.zkProvider = nil
	}
}

func (l *Loader) SetOnChange(callback func(*Config) error) {
	l.options.OnChange = callback
}

func LoadConfig(opts LoaderOptions) (*Config, error) {
	cfg, _, err := LoadConfigWithLoader(opts)
	return cfg, err
}

func LoadConfigWithLoader(opts LoaderOptions) (*Config, *Loader, error) {
	loader, err := NewLoader(opts)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create loader: %w", err)
	}

	cfg, err := loader.Load()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load config: %w", err)
	}

	return cfg, loader, nil
}

func ParseConfigType(s string) (ConfigType, error) {
	s = strings.ToLower(strings.TrimSpace(s))

	switch s {
	case "file":
		return ConfigTypeFile, nil
	case "consul":
		return ConfigTypeConsul, nil
	case "etcd":
		return ConfigTypeEtcd, nil
	case "zookeeper", "zk":
		return ConfigTypeZookeeper, nil
	default:
		return "", fmt.Errorf("invalid config type: %s (valid types: file, consul, etcd, zookeeper)", s)
	}
}
