# Testing Guide

This guide covers testing practices, strategies, and tools used in Hector development.

## Overview

Hector follows **proper unit testing best practices** to ensure code quality, reliability, and maintainability. Our testing approach focuses on:

- **Test Behavior, Not Implementation** - Tests verify what the code does, not how it does it
- **Fast, Independent, Repeatable** - Tests run quickly and consistently
- **Comprehensive Coverage** - Critical paths and edge cases are thoroughly tested
- **Clear, Descriptive Tests** - Test names and structure are self-documenting

## Testing Philosophy

### Unit Testing Best Practices

1. **Single Responsibility per Test**
   - Each test focuses on one specific behavior
   - Clear test names describing the scenario being tested

2. **Table-Driven Tests**
   - Multiple scenarios tested efficiently
   - Easy to add new test cases
   - Consistent test structure

3. **Comprehensive Edge Case Coverage**
   - Boundary conditions, empty inputs, error conditions
   - Invalid data, network failures, timeouts
   - Real-world scenarios and edge cases

4. **Proper Mocking & Dependency Injection**
   - Use `httptest.Server` for HTTP testing
   - Temporary files/directories for file I/O testing
   - Environment variable manipulation for configuration testing
   - Avoid over-abstraction or complex mocking frameworks

5. **Fast, Independent, Repeatable**
   - No shared state between tests
   - Deterministic results
   - No external dependencies

## Test Structure

### Package Organization

```
pkg/
├── package_name/
│   ├── package.go          # Implementation
│   ├── package_test.go     # Unit tests
│   └── ...
```

### Test File Naming

- `*_test.go` - Standard Go test files
- `*_comprehensive_test.go` - Comprehensive test suites
- `*_integration_test.go` - Integration tests (if needed)

### Test Function Naming

```go
// Good: Clear, descriptive test names
func TestConfig_Validate_ValidMinimalConfig(t *testing.T)
func TestHTTPClient_Do_RateLimitWithRetryAfter(t *testing.T)
func TestAgentConfig_SetDefaults_EmptyConfig(t *testing.T)

// Good: Table-driven test structure
func TestLLMProviderConfig_Validate(t *testing.T) {
    tests := []struct {
        name    string
        config  LLMProviderConfig
        wantErr bool
    }{
        {
            name: "valid_openai_config",
            config: LLMProviderConfig{...},
            wantErr: false,
        },
        // ... more test cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

## Testing Tools & Commands

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run tests for specific package
go test ./pkg/config/...

# Run tests with verbose output
go test -v ./pkg/config/...

# Run tests with coverage report
go test -coverprofile=coverage.out ./pkg/config/...
go tool cover -html=coverage.out
```

### Test Coverage

```bash
# Generate coverage report
make test-coverage

# View coverage summary
make test-coverage-summary

# Generate HTML coverage report
go tool cover -html=coverage.out -o coverage.html
```

### Quality Checks

```bash
# Run all quality checks (fmt, vet, lint, test)
make quality

# Run full CI simulation (deps + fmt + vet + lint + test + build)
make pre-commit

# Format code
make fmt

# Run static analysis
make vet

# Run linter (auto-installs golangci-lint if needed)
make lint

# Run tests with race detection
make test-race
```

## Test Categories

### 1. Unit Tests

**Purpose**: Test individual components in isolation

**Examples**:
- Configuration validation and default setting
- HTTP client retry logic and rate limiting
- Tool parameter validation and execution
- Agent configuration and behavior

**Best Practices**:
- Mock external dependencies (HTTP clients, file systems)
- Use table-driven tests for multiple scenarios
- Test both success and error paths
- Verify edge cases and boundary conditions

### 2. Integration Tests

**Purpose**: Test component interactions and system behavior

**Examples**:
- A2A protocol communication
- Agent-to-agent interactions
- End-to-end workflow execution
- Configuration loading and validation

**Best Practices**:
- Use test servers and temporary resources
- Test realistic scenarios
- Verify system behavior under various conditions
- Clean up resources after tests

### 3. Performance Tests

**Purpose**: Verify performance characteristics and limits

**Examples**:
- HTTP client retry performance
- Configuration loading speed
- Memory usage under load
- Concurrent access patterns

## Testing Patterns

### Configuration Testing

```go
func TestConfig_Validate(t *testing.T) {
    tests := []struct {
        name    string
        config  *Config
        wantErr bool
    }{
        {
            name: "valid_minimal_config",
            config: &Config{
                Agents: map[string]AgentConfig{
                    "test-agent": {
                        Name: "Test Agent",
                        LLM:  "test-llm",
                    },
                },
                LLMs: map[string]LLMProviderConfig{
                    "test-llm": {
                        Type:   "openai",
                        Model:  "gpt-4o",
                        Host:   "https://api.openai.com/v1",
                        APIKey: "sk-test-key",
                    },
                },
            },
            wantErr: false,
        },
        // ... more test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            tt.config.SetDefaults()
            err := tt.config.Validate()
            if (err != nil) != tt.wantErr {
                t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### HTTP Client Testing

```go
func TestClient_Do_RateLimitWithRetryAfter(t *testing.T) {
    attemptCount := 0
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        attemptCount++
        if attemptCount == 1 {
            w.Header().Set("Retry-After", "1")
            w.WriteHeader(http.StatusTooManyRequests)
        } else {
            w.WriteHeader(http.StatusOK)
            w.Write([]byte("success after rate limit"))
        }
    }))
    defer server.Close()

    client := New(
        WithHTTPClient(server.Client()),
        WithMaxRetries(3),
        WithHeaderParser(ParseOpenAIRateLimitHeaders),
    )
    req, _ := http.NewRequest("GET", server.URL, nil)

    start := time.Now()
    resp, err := client.Do(req)
    duration := time.Since(start)

    if err != nil {
        t.Fatalf("Do() error = %v, want nil", err)
    }
    if resp.StatusCode != http.StatusOK {
        t.Errorf("Do() status code = %d, want %d", resp.StatusCode, http.StatusOK)
    }
    if attemptCount != 2 {
        t.Errorf("Expected 2 attempts, got %d", attemptCount)
    }
    // Should have waited at least 1 second for Retry-After
    if duration < 1*time.Second {
        t.Errorf("Expected to wait at least 1s, waited %v", duration)
    }
}
```

### File I/O Testing

```go
func TestFileWriterTool_WithTempDir(t *testing.T) {
    tests := []struct {
        name           string
        filePath       string
        content        string
        expectedResult string
    }{
        {
            name:           "create_new_file",
            filePath:       "test.txt",
            content:        "Hello, World!",
            expectedResult: "created",
        },
        {
            name:           "overwrite_existing_file",
            filePath:       "test.txt",
            content:        "Updated content",
            expectedResult: "overwritten",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Create temporary directory
            tempDir := t.TempDir()
            
            // Create tool with temp directory
            tool := NewFileWriterTool(&config.FileWriterConfig{
                WorkingDirectory: tempDir,
                MaxFileSize:      1024,
            })

            // Test file operations
            result := tool.Execute(map[string]interface{}{
                "path":    tt.filePath,
                "content": tt.content,
            })

            if !strings.Contains(result.Content, tt.expectedResult) {
                t.Errorf("Expected result to contain '%s', got: %s", tt.expectedResult, result.Content)
            }
        })
    }
}
```

### Environment Variable Testing

```go
func TestLoadConfig_EnvironmentVariableExpansion(t *testing.T) {
    // Set up environment variables
    os.Setenv("OPENAI_API_KEY", "sk-test-key-123")
    defer os.Unsetenv("OPENAI_API_KEY")

    yaml := `
llms:
  test-llm:
    type: "openai"
    model: "gpt-4o"
    host: "https://api.openai.com/v1"
    api_key: "${OPENAI_API_KEY}"
`

    config, err := LoadConfigFromString(yaml)
    if err != nil {
        t.Fatalf("LoadConfigFromString() error = %v", err)
    }

    if llm, exists := config.LLMs["test-llm"]; exists {
        if llm.APIKey != "sk-test-key-123" {
            t.Errorf("API key = %v, want %v", llm.APIKey, "sk-test-key-123")
        }
    } else {
        t.Error("Expected LLM 'test-llm' to exist")
    }
}
```

## Test Coverage Goals

### Current Coverage Status

- **HTTPClient Package**: 99.0% coverage
- **Config Package**: 56.6% coverage
- **Tools Package**: 57.5% coverage
- **Registry Package**: 100.0% coverage
- **Reasoning Package**: 18.8% coverage

### Coverage Targets

- **Critical Packages**: >90% coverage (HTTPClient, Config, Agent)
- **Core Packages**: >80% coverage (Tools, LLMs, A2A)
- **Utility Packages**: >70% coverage (Registry, Utils)
- **Overall Project**: >75% coverage

## CI/CD Integration

### GitHub Actions

Tests and quality checks are automatically run in CI/CD pipeline:

```yaml
# .github/workflows/ci.yml
- name: Run tests
  run: make test

- name: Run tests with coverage
  run: make test-coverage

- name: Run go vet
  run: make vet

- name: Run golangci-lint
  uses: golangci/golangci-lint-action@v6
  with:
    version: v1.55.2
    args: --timeout=5m --verbose

- name: Upload coverage reports
  uses: codecov/codecov-action@v4
  with:
    file: ./coverage.out
```

### Pre-commit Hooks

```bash
# Run before committing (full CI simulation)
make pre-commit  # deps + fmt + vet + lint + test + build

# Or for faster feedback during development
make quality     # fmt + vet + lint + test
```

## Common Testing Patterns

### 1. Validation Testing

```go
func TestConfig_Validate(t *testing.T) {
    // Test valid configurations
    // Test invalid configurations
    // Test edge cases
    // Test error messages
}
```

### 2. Default Setting Testing

```go
func TestConfig_SetDefaults(t *testing.T) {
    // Test empty configurations get defaults
    // Test partial configurations preserve values
    // Test zero values get defaults
    // Test environment variable integration
}
```

### 3. Error Handling Testing

```go
func TestConfig_ErrorHandling(t *testing.T) {
    // Test invalid input handling
    // Test network error handling
    // Test timeout handling
    // Test retry logic
}
```

### 4. Integration Testing

```go
func TestAgent_EndToEnd(t *testing.T) {
    // Test complete agent workflow
    // Test agent-to-agent communication
    // Test tool execution
    // Test session management
}
```

## Best Practices Summary

### ✅ Do

- Write tests that verify behavior, not implementation
- Use descriptive test names that explain the scenario
- Use table-driven tests for multiple similar scenarios
- Test both success and error paths
- Use temporary resources (files, directories, servers)
- Mock external dependencies appropriately
- Clean up resources after tests
- Test edge cases and boundary conditions
- Verify error messages and types
- Test concurrent access patterns where relevant

### ❌ Don't

- Test private implementation details
- Create overly complex mocking frameworks
- Share state between tests
- Skip error handling in tests
- Test third-party library functionality
- Create tests that depend on external services
- Write tests that are slow or flaky
- Test configuration that's not realistic
- Ignore test failures or warnings

## Troubleshooting

### Common Issues

1. **Tests failing due to environment variables**
   - Use `t.Setenv()` or `os.Setenv()` with `defer os.Unsetenv()`
   - Set up environment in test setup, clean up in teardown

2. **File I/O tests failing**
   - Use `t.TempDir()` for temporary directories
   - Ensure proper file permissions and paths

3. **HTTP tests timing out**
   - Use `httptest.Server` for controlled HTTP responses
   - Set appropriate timeouts for test scenarios

4. **Tests not deterministic**
   - Avoid time-dependent logic without mocking
   - Use fixed timestamps or mock time functions
   - Ensure proper cleanup of shared resources

### Debugging Tests

```bash
# Run specific test with verbose output
go test -v -run TestConfig_Validate ./pkg/config/

# Run test with race detection
go test -race ./pkg/config/

# Run test with coverage for specific function
go test -coverprofile=coverage.out -covermode=count ./pkg/config/
go tool cover -func=coverage.out | grep Validate
```

## Resources

- [Go Testing Documentation](https://golang.org/pkg/testing/)
- [Effective Go - Testing](https://golang.org/doc/effective_go.html#testing)
- [Go Testing Best Practices](https://github.com/golang/go/wiki/TestComments)
- [Table-Driven Tests](https://github.com/golang/go/wiki/TableDrivenTests)
- [Advanced Testing in Go](https://about.sourcegraph.com/go/advanced-testing-in-go)

---

**Testing is not just about finding bugs—it's about building confidence in your code and ensuring it behaves correctly under all conditions.**
