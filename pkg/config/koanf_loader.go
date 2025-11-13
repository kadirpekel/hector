package config

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/consul"
	"github.com/knadh/koanf/providers/etcd"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
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
	koanf    *koanf.Koanf
	options  LoaderOptions
	parser   *yaml.YAML
	stopChan chan struct{}
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

	return &Loader{
		koanf:    koanf.New("."),
		options:  opts,
		parser:   yaml.Parser(),
		stopChan: make(chan struct{}),
	}, nil
}

func (l *Loader) Load() (*Config, error) {
	var provider koanf.Provider
	var err error

	switch l.options.Type {
	case ConfigTypeFile:
		provider = file.Provider(l.options.Path)

	case ConfigTypeConsul:

		consulConfig := api.DefaultConfig()
		consulConfig.Address = l.options.Endpoints[0]

		provider = consul.Provider(consul.Config{
			Cfg: consulConfig,
			Key: l.options.Path,
		})

	case ConfigTypeEtcd:

		provider = etcd.Provider(etcd.Config{
			Endpoints:   l.options.Endpoints,
			DialTimeout: 5 * time.Second,
			Key:         l.options.Path,
		})

	case ConfigTypeZookeeper:

		zkProvider, err := NewZookeeperProvider(l.options.Endpoints, l.options.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to create zookeeper provider: %w", err)
		}
		provider = zkProvider

	default:
		return nil, fmt.Errorf("unsupported config type: %s", l.options.Type)
	}

	var parser koanf.Parser
	if l.options.Type == ConfigTypeFile || l.options.Type == ConfigTypeZookeeper {
		parser = l.parser
	} else {
		parser = nil
	}

	if err = l.koanf.Load(provider, parser); err != nil {
		return nil, fmt.Errorf("failed to load config from %s: %w", l.options.Type, err)
	}

	if err := l.expandEnvVarsInKoanf(); err != nil {
		return nil, fmt.Errorf("failed to expand environment variables: %w", err)
	}

	cfg, err := l.unmarshalAndProcess()
	if err != nil {
		return nil, err
	}

	if l.options.Watch {
		go l.watch(provider)
	}

	return cfg, nil
}

type Watcher interface {
	Watch(cb func(event interface{}, err error)) error
}

func (l *Loader) watch(provider koanf.Provider) {

	watcher, ok := provider.(Watcher)
	if !ok {
		log.Printf("⚠️  Provider %s does not support watching", l.options.Type)
		return
	}

	log.Printf("🔄 Config watcher started for %s (reactive watch via koanf)", l.options.Type)

	err := watcher.Watch(func(event interface{}, err error) {

		select {
		case <-l.stopChan:
			log.Printf("🛑 Config watcher stopped for %s", l.options.Type)
			return
		default:
		}

		if err != nil {
			log.Printf("⚠️  Watch error: %v", err)
			return
		}

		var parser koanf.Parser
		if l.options.Type == ConfigTypeFile || l.options.Type == ConfigTypeZookeeper {
			parser = l.parser
		} else {
			parser = nil
		}

		if err := l.koanf.Load(provider, parser); err != nil {
			log.Printf("⚠️  Failed to reload config: %v", err)
			return
		}

		if err := l.expandEnvVarsInKoanf(); err != nil {
			log.Printf("⚠️  Failed to expand env vars in reloaded config: %v", err)
			return
		}

		newCfg, err := l.unmarshalAndProcess()
		if err != nil {
			log.Printf("⚠️  Reloaded config processing failed: %v", err)
			return
		}

		if l.options.OnChange != nil {
			if err := l.options.OnChange(newCfg); err != nil {
				log.Printf("⚠️  Config change callback failed: %v", err)
			} else {
				log.Printf("✅ Configuration reloaded successfully from %s", l.options.Type)
			}
		} else {
			log.Printf("⚠️  Config change detected but OnChange callback not set - config reloaded but server not notified")
		}
	})

	if err != nil {
		log.Printf("⚠️  Watch stopped with error: %v", err)
	}
}

func (l *Loader) unmarshalAndProcess() (*Config, error) {
	// First, perform strict validation to catch typos and unknown fields
	strictResult, err := ValidateConfigStructure(l.koanf)
	if err != nil {
		return nil, fmt.Errorf("strict validation check failed: %w", err)
	}

	if !strictResult.Valid() {
		// Return validation errors as a formatted error
		return nil, fmt.Errorf("configuration has structural errors:\n%s", strictResult.FormatErrors())
	}

	// Now do the normal unmarshal (we know it will succeed structurally)
	cfg := &Config{}
	if err := l.koanf.UnmarshalWithConf("", cfg, koanf.UnmarshalConf{
		Tag: "yaml",
	}); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	processedCfg, err := ProcessConfigPipeline(cfg)
	if err != nil {
		return nil, fmt.Errorf("config processing failed: %w", err)
	}

	return processedCfg, nil
}

func (l *Loader) expandEnvVarsInKoanf() error {
	rawMap := l.koanf.Raw()

	expandedMap := ExpandEnvVarsInData(rawMap)

	expandedMapData, ok := expandedMap.(map[string]interface{})
	if !ok {
		return fmt.Errorf("unexpected type after env var expansion")
	}

	newKoanf := koanf.New(".")
	if err := newKoanf.Load(confmap.Provider(expandedMapData, "."), nil); err != nil {
		return fmt.Errorf("failed to load expanded config: %w", err)
	}

	l.koanf = newKoanf

	return nil
}

func (l *Loader) Stop() {
	close(l.stopChan)
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
