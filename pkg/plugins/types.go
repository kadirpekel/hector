package plugins

import (
	"context"
	"fmt"
)

type PluginType string

const (
	PluginTypeLLM            PluginType = "llm_provider"
	PluginTypeDatabase       PluginType = "database_provider"
	PluginTypeEmbedder       PluginType = "embedder_provider"
	PluginTypeTool           PluginType = "tool_provider"
	PluginTypeReasoning      PluginType = "reasoning_strategy"
	PluginTypeDocumentParser PluginType = "document_parser"
)

type PluginProtocol string

const (
	ProtocolGRPC PluginProtocol = "grpc"
)

type PluginStatus string

const (
	StatusUnloaded   PluginStatus = "unloaded"
	StatusLoading    PluginStatus = "loading"
	StatusReady      PluginStatus = "ready"
	StatusError      PluginStatus = "error"
	StatusCrashed    PluginStatus = "crashed"
	StatusShutdown   PluginStatus = "shutdown"
	StatusRestarting PluginStatus = "restarting"
)

type PluginManifest struct {
	Name             string                 `yaml:"name" json:"name"`
	Version          string                 `yaml:"version" json:"version"`
	Author           string                 `yaml:"author" json:"author"`
	Description      string                 `yaml:"description" json:"description"`
	Type             PluginType             `yaml:"type" json:"type"`
	Protocol         PluginProtocol         `yaml:"protocol" json:"protocol"`
	HectorVersion    string                 `yaml:"hector_version" json:"hector_version"`
	ConfigSchema     PluginConfigSchema     `yaml:"config_schema" json:"config_schema"`
	Capabilities     map[string]interface{} `yaml:"capabilities" json:"capabilities"`
	Homepage         string                 `yaml:"homepage,omitempty" json:"homepage,omitempty"`
	License          string                 `yaml:"license,omitempty" json:"license,omitempty"`
	RepositoryURL    string                 `yaml:"repository_url,omitempty" json:"repository_url,omitempty"`
	DocumentationURL string                 `yaml:"documentation_url,omitempty" json:"documentation_url,omitempty"`
}

type PluginConfigSchema struct {
	Required []string               `yaml:"required" json:"required"`
	Optional []string               `yaml:"optional" json:"optional"`
	Defaults map[string]interface{} `yaml:"defaults,omitempty" json:"defaults,omitempty"`
}

type PluginConfig struct {
	Name     string                 `yaml:"name" json:"name"`
	Type     PluginProtocol         `yaml:"type" json:"type"`
	Path     string                 `yaml:"path" json:"path"`
	Enabled  bool                   `yaml:"enabled" json:"enabled"`
	Config   map[string]interface{} `yaml:"config" json:"config"`
	Manifest *PluginManifest        `yaml:"-" json:"-"`
}

type Plugin interface {
	Initialize(ctx context.Context, config map[string]interface{}) error

	Shutdown(ctx context.Context) error

	GetManifest() *PluginManifest

	GetStatus() PluginStatus

	Health(ctx context.Context) error
}

type PluginLoader interface {
	Load(ctx context.Context, config *PluginConfig) (Plugin, error)

	Unload(ctx context.Context, plugin Plugin) error

	SupportedProtocol() PluginProtocol

	Validate(ctx context.Context, path string) error
}

type PluginError struct {
	PluginName string
	Operation  string
	Message    string
	Err        error
}

func (e *PluginError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[Plugin:%s] %s failed: %s: %v", e.PluginName, e.Operation, e.Message, e.Err)
	}
	return fmt.Sprintf("[Plugin:%s] %s failed: %s", e.PluginName, e.Operation, e.Message)
}

func (e *PluginError) Unwrap() error {
	return e.Err
}

func NewPluginError(pluginName, operation, message string, err error) *PluginError {
	return &PluginError{
		PluginName: pluginName,
		Operation:  operation,
		Message:    message,
		Err:        err,
	}
}

var (
	ErrPluginNotFound      = fmt.Errorf("plugin not found")
	ErrPluginNotLoaded     = fmt.Errorf("plugin not loaded")
	ErrPluginAlreadyLoaded = fmt.Errorf("plugin already loaded")
	ErrPluginCrashed       = fmt.Errorf("plugin crashed")
	ErrPluginTimeout       = fmt.Errorf("plugin operation timed out")
	ErrInvalidManifest     = fmt.Errorf("invalid plugin manifest")
	ErrIncompatibleVersion = fmt.Errorf("incompatible plugin version")
	ErrUnsupportedProtocol = fmt.Errorf("unsupported plugin protocol")
	ErrInvalidConfig       = fmt.Errorf("invalid plugin configuration")
)

type PluginLifecycleHook func(ctx context.Context, plugin Plugin) error

type PluginLifecycleHooks struct {
	BeforeLoad   PluginLifecycleHook
	AfterLoad    PluginLifecycleHook
	BeforeInit   PluginLifecycleHook
	AfterInit    PluginLifecycleHook
	BeforeUnload PluginLifecycleHook
	AfterUnload  PluginLifecycleHook
	OnCrash      PluginLifecycleHook
}
