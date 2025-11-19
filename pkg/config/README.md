# Configuration Package

This package provides configuration loading with support for multiple backends.

## Architecture Overview

The configuration system is built on a **unified processing pipeline** that ensures consistency across all loading paths (file, Consul, etcd, ZooKeeper, zero-config).

### Core Principle: Single Pipeline

**All configuration MUST flow through the same processing pipeline:**

```
┌─────────────────────────────────────────────────────────────┐
│                  Configuration Pipeline                      │
└─────────────────────────────────────────────────────────────┘

Phase 1: LOAD
  ├─ Source: File / Consul / Etcd / ZooKeeper / Zero-Config
  ├─ Parser: YAML/JSON (direct parsing)
  └─ Output: Raw Config struct

Phase 2: EXPAND
  ├─ Environment variable expansion (${VAR}, ${VAR:-default})
  ├─ Shortcut expansion (docs_folder → document_stores)
  └─ Reference resolution

Phase 3: DEFAULT
  ├─ Set sensible defaults for optional fields
  ├─ Create zero-config defaults if needed
  └─ Apply provider-specific defaults

Phase 4: VALIDATE
  ├─ Required fields present
  ├─ Valid values and ranges
  └─ Cross-field consistency
```

### Loader

**`loader.go`** - **Unified Configuration Loader**
   - Supports: File, Consul, Etcd, ZooKeeper
   - Features: Reactive watching, environment variable expansion
   - **Always processes through the pipeline**
   - Used by: All parts of the system (CLI, runtime, tests)

### Providers

- **File**: Direct file reading with YAML/JSON parser (YAML standard)
- **Consul**: Direct Consul API with YAML/JSON parser (YAML standard)
- **Etcd**: Direct Etcd client with YAML/JSON parser (YAML standard)
- **ZooKeeper**: Custom provider with YAML/JSON parser (YAML standard)

**Note**: All providers support YAML as the standard format. JSON is supported as a fallback for compatibility.

### Configuration Flow

**Production (File/Consul/Etcd/ZooKeeper):**
```
CLI Command / Runtime
    ↓
loader.go: LoadConfig()
    ↓
Provider → Parser → Raw Config
    ↓
expandEnvVarsInConfig() [Phase 1: Env expansion]
    ↓
ProcessConfigPipeline() [Phase 2-4: Expand, Default, Validate]
    ↓
Valid, Ready-to-Use Config
    ↓
Runtime
```

**Zero-Config (CLI Flags):**
```
CLI Command / Runtime
    ↓
CreateZeroConfig() → Raw Config
    ↓
ProcessConfigPipeline() [Phase 2-4: SAME as file-based!]
    ↓
Valid, Ready-to-Use Config
    ↓
Runtime
```

**Key Insight:** Both paths converge at `ProcessConfigPipeline()`, ensuring 100% consistency.

## Usage

### Load from File (YAML)

```go
cfg, loader, err := config.LoadConfig(config.LoaderOptions{
    Type: config.ConfigTypeFile,
    Path: "configs/production.yaml",
    Watch: true,
})
defer loader.Stop() // Stop watcher on exit
```

### Load from Consul (JSON)

```go
cfg, loader, err := config.LoadConfig(config.LoaderOptions{
    Type: config.ConfigTypeConsul,
    Path: "hector/production",
    Endpoints: []string{"consul.example.com:8500"},
    Watch: true,
    OnChange: func(newCfg *config.Config) error {
        // Handle config reload
        return nil
    },
})
```

### Load from Etcd (JSON)

```go
cfg, loader, err := config.LoadConfig(config.LoaderOptions{
    Type: config.ConfigTypeEtcd,
    Path: "/hector/production",
    Endpoints: []string{"etcd1:2379", "etcd2:2379"},
    Watch: true,
})
```

### Load from ZooKeeper (YAML)

```go
cfg, loader, err := config.LoadConfig(config.LoaderOptions{
    Type: config.ConfigTypeZookeeper,
    Path: "/hector/production",
    Endpoints: []string{"zk1:2181", "zk2:2181"},
    Watch: true,
})
```

## Format Differences

### File & ZooKeeper: YAML

```yaml
version: "1.0"
llms:
  openai:
    type: openai
    model: gpt-4
    api_key: ${OPENAI_API_KEY}  # Env vars supported
```

### Consul & Etcd: JSON

```json
{
  "version": "1.0",
  "llms": {
    "openai": {
      "type": "openai",
      "model": "gpt-4",
      "api_key": "${OPENAI_API_KEY}"
    }
  }
}
```

**Note**: JSON doesn't support comments. Use YAML for development, JSON for KV stores.

## Environment Variable Expansion

Both YAML and JSON configs support environment variable expansion using `${VAR_NAME}` syntax:

```yaml
api_key: ${OPENAI_API_KEY}
```

This is expanded at load time before validation.

## Watching & Reloading

All backends support reactive watching:

- **File**: File system watching
- **Consul**: Consul blocking queries (instant)
- **Etcd**: Etcd watch API (instant)
- **ZooKeeper**: ZooKeeper watch mechanism (instant)

When a change is detected:
1. Configuration reloaded from backend
2. Validated
3. If valid: OnChange callback called
4. If invalid: Error logged, current config retained

## Testing

```bash
# Run config tests
go test ./pkg/config/...

# Test with real providers
docker-compose -f deployments/docker-compose.config-providers.yaml up -d
./scripts/test-config-providers.sh consul --watch
```

## Files

| File | Purpose |
|------|---------|
| `loader.go` | Main distributed config loader |
| `zookeeper_provider.go` | Custom ZooKeeper provider |
| `config.go` | Main Config struct and validation |
| `types.go` | All configuration types |
| `env.go` | Environment variable expansion |
| `interface.go` | Configuration interfaces |

## Design Principles

### 1. Single Source of Truth
**`ProcessConfigPipeline()`** is the ONLY entry point for configuration processing.
- File loading → Pipeline
- Zero-config → Pipeline
- Tests → Pipeline
- Config watching/reloading → Pipeline

### 2. Clear Phase Separation
Each phase has a single responsibility:
- **EXPAND** - Transforms shortcuts and resolves references
- **DEFAULT** - Fills in missing values
- **VALIDATE** - Catches errors

### 3. No Side Effects
- `CreateZeroConfig()` returns RAW config (no defaults, no validation)
- `LoadConfig()` processes through pipeline automatically
- Pipeline is the transformation boundary

### 4. Consistency Guarantees
Because all paths use the same pipeline:
- Environment variables expanded the same way
- Shortcuts work identically
- Validation rules are consistent
- Defaults applied uniformly

## Migration Notes

### Before (Inconsistent)
```go
// Path 1: File loading
cfg, _ := LoadConfig(opts)
cfg.SetDefaults()  // Sometimes called
cfg.Validate()     // Sometimes called

// Path 2: Zero-config
cfg := CreateZeroConfig(opts)
cfg.SetDefaults()  // Called internally - different timing!

// Path 3: Tests
cfg := loadConfigForTest(path)  // Custom logic!
```

### After (Unified)
```go
// All paths use the SAME pipeline
cfg, loader, err := config.LoadConfig(config.LoaderOptions{
    Type: config.ConfigTypeFile,
    Path: "config.yaml",
    Watch: true,
})
defer loader.Stop()

// Or for zero-config:
rawCfg := config.CreateZeroConfig(opts)
cfg, err := config.ProcessConfigPipeline(rawCfg)

// Tests use production code:
// loadConfigForTest() calls LoadConfig() - same pipeline!
```

