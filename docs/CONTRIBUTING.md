---
layout: default
title: Contributing
nav_order: 30
description: "How to contribute to Hector development"
---

# Contributing to Hector

## Development Setup

### Prerequisites

- Go 1.24+
- Git

### Setup

```bash
# Fork and clone
git clone https://github.com/your-username/hector.git
cd hector

# Install dependencies
go mod download

# Build
make build

# Run tests
make test
```

---

## Development Workflow

```bash
# Create feature branch
git checkout -b feature/your-feature

# Make changes, add tests

# Run quality checks
make quality  # fmt + vet + lint + test

# Commit with conventional format
git commit -m "feat: your feature description"

# Push and create PR
git push origin feature/your-feature
```

### Commit Message Format

Follow [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` - New features
- `fix:` - Bug fixes
- `docs:` - Documentation
- `test:` - Tests
- `refactor:` - Code refactoring
- `chore:` - Maintenance

---

## Code Standards

### Quality Checks

All code must pass:

```bash
# Format code
make fmt

# Static analysis
make vet

# Linting (auto-installs golangci-lint)
make lint

# Tests
make test

# All checks
make quality

# Full CI simulation
make pre-commit
```

**Requirements:**
- ✅ `gofmt` formatted
- ✅ `go vet` passes
- ✅ `golangci-lint` passes
- ✅ All tests pass
- ✅ Builds successfully

### Go Style

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Meaningful names
- Comment exported functions/types
- Keep functions focused

---

## Testing

### Writing Tests

**Requirements:**
- Test all new functionality
- Follow table-driven test pattern
- Test success and error cases
- Include edge cases
- No external dependencies

**Test Structure:**

```go
func TestComponent_Behavior(t *testing.T) {
    tests := []struct {
        name    string
        input   Input
        want    Output
        wantErr bool
    }{
        {
            name: "valid_case",
            input: Input{...},
            want: Output{...},
            wantErr: false,
        },
        // More cases...
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Component.Behavior(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
            }
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Running Tests

```bash
# All tests
make test

# With coverage
make test-coverage

# Specific package
go test ./pkg/config/...

# With race detection
make test-race

# Verbose
go test -v ./pkg/config/...
```

### Coverage Targets

- Critical packages (Config, Agent, HTTPClient): >90%
- Core packages (Tools, LLMs, A2A): >80%
- Utility packages: >70%

### Testing Patterns

**Use `httptest.Server` for HTTP:**
```go
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
}))
defer server.Close()
```

**Use `t.TempDir()` for files:**
```go
tempDir := t.TempDir()
// Automatically cleaned up
```

**Use `t.Setenv()` for env vars:**
```go
t.Setenv("API_KEY", "test-key")
```

---

## Documentation

### User Documentation

User-facing docs in `docs/`:

```bash
cd docs
bundle install
bundle exec jekyll serve --livereload
# Open http://localhost:4000
```

### Code Documentation

- Update README.md for user-facing changes
- Add package docs for new packages
- Update API docs for interface changes

---

## Project Structure

```
hector/
├── cmd/hector/    # CLI application
├── pkg/           # Public packages
│   ├── a2a/       # A2A protocol
│   ├── agent/     # Core agent
│   ├── config/    # Configuration
│   ├── llms/      # LLM providers
│   ├── tools/     # Built-in tools
│   └── ...
├── docs/          # Documentation
└── configs/       # Example configurations
```

---

## License

By contributing, you agree that your contributions will be licensed under the AGPL-3.0 license.
