# Contributing to Hector

Thank you for your interest in contributing to Hector! Since we're in alpha, this is a great time to help shape the project.

## Getting Started

### Prerequisites

- Go 1.24 or later
- Git
- Basic understanding of the A2A protocol

### Development Setup

1. **Fork and clone the repository**
   ```bash
   git clone https://github.com/your-username/hector.git
   cd hector
   ```

2. **Install dependencies**
   ```bash
   go mod download
   ```

3. **Build the project**
   ```bash
   make build
   # or
   go build -o hector ./cmd/hector
   ```

4. **Run tests**
   ```bash
   make test
   # or
   go test ./...
   ```

## Development Workflow

### Making Changes

1. **Create a feature branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes**
   - Follow Go conventions
   - Add tests for new functionality
   - Update documentation as needed

3. **Run quality checks**
   ```bash
   make pre-commit  # Runs all CI checks (deps, fmt, vet, lint, test, build)
   # or for development
   make quality     # Runs fmt, vet, lint, and test
   ```

4. **Commit your changes**
   ```bash
   git add .
   git commit -m "feat: add your feature description"
   ```

5. **Push and create a PR**
   ```bash
   git push origin feature/your-feature-name
   ```

### Commit Message Format

We follow [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` - New features
- `fix:` - Bug fixes
- `docs:` - Documentation changes
- `style:` - Code style changes (formatting, etc.)
- `refactor:` - Code refactoring
- `test:` - Adding or updating tests
- `chore:` - Maintenance tasks

Examples:
```
feat: add support for custom reasoning strategies
fix: handle empty agent responses gracefully
docs: update API documentation
test: add integration tests for A2A protocol
```

## Code Standards

### Go Code Style

- Use `gofmt` for formatting
- Follow [Effective Go](https://golang.org/doc/effective_go.html) guidelines
- Use meaningful variable and function names
- Add comments for exported functions and types
- Keep functions focused and small
- Pass all linting checks with `golangci-lint`

### Code Quality Checks

All code must pass our comprehensive quality checks:

```bash
# Run all quality checks
make quality

# Run individual checks
make fmt      # Format code with gofmt
make vet      # Run go vet for static analysis
make lint     # Run golangci-lint (auto-installs if needed)
make test     # Run all tests
make build    # Build the project
```

**Quality Requirements**:
- ✅ Code must be formatted with `gofmt`
- ✅ Must pass `go vet` static analysis
- ✅ Must pass `golangci-lint` with zero warnings
- ✅ All tests must pass
- ✅ Project must build successfully
- ✅ No race conditions (use `make test-race`)

### Testing

Hector follows **proper unit testing best practices**. All contributions must include comprehensive tests.

**Requirements**:
- Write tests for all new functionality
- Maintain or improve test coverage
- Follow our [Testing Guide](TESTING.md) for best practices
- Use table-driven tests for multiple scenarios
- Test both success and error cases
- Include edge cases and boundary conditions

**Test Coverage Targets**:
- Critical packages (Config, Agent, HTTPClient): >90%
- Core packages (Tools, LLMs, A2A): >80%
- Utility packages: >70%

**Running Tests**:
```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run quality checks (fmt + vet + lint + test)
make quality

# Run full CI simulation (deps + fmt + vet + lint + test + build)
make pre-commit
```

See [TESTING.md](TESTING.md) for comprehensive testing guidelines.

### Documentation

- Update README.md for user-facing changes
- Add package documentation for new packages
- Update API documentation for interface changes
- Include examples in documentation

## Project Structure

```
hector/
├── cmd/hector/    # CLI application
├── pkg/           # Public API packages
│   ├── a2a/       # A2A protocol implementation
│   ├── agent/     # Core agent implementation
│   ├── config/    # Configuration management
│   ├── llms/      # LLM providers
│   ├── tools/     # Built-in tools
│   ├── reasoning/ # Reasoning strategies
│   └── ...        # Other public packages
├── internal/      # Private packages
├── docs/          # Documentation
├── examples/      # Example configurations
└── ...
```

## Areas for Contribution

### High Priority

- **A2A Protocol Compliance**: Ensure full compliance with the A2A specification
- **Documentation**: Improve guides, examples, and API docs
- **Testing**: Improve test coverage for core packages (see [TESTING.md](TESTING.md))
- **Code Quality**: Maintain high code quality standards and linting compliance
- **Performance**: Optimize agent execution and memory usage

### Medium Priority

- **New Tools**: Implement additional built-in tools
- **LLM Providers**: Add support for more LLM providers
- **Database Integrations**: Add more vector database options
- **Plugin System**: Enhance the gRPC plugin system

### Low Priority

- **UI/UX**: Web interface for agent management
- **Monitoring**: Metrics and observability
- **Security**: Enhanced security features

## Alpha Status Considerations

Since Hector is in alpha:

- **APIs may change** - We're still refining interfaces
- **Breaking changes** - Will be documented in release notes
- **Experimental features** - May be removed or modified
- **Feedback is valuable** - Your input helps shape the project

## Getting Help

- **GitHub Issues**: Report bugs or request features
- **GitHub Discussions**: Ask questions or discuss ideas
- **Documentation**: Check the [docs/](docs/) directory

## Release Process

Releases are managed through:

1. **Semantic Versioning**: We follow [SemVer](https://semver.org/)
2. **GitHub Releases**: Automated via GoReleaser
3. **Alpha Releases**: Pre-release versions for testing

## License

By contributing to Hector, you agree that your contributions will be licensed under the AGPL-3.0 license.

## Code of Conduct

We expect all contributors to:

- Be respectful and inclusive
- Focus on constructive feedback
- Help others learn and grow
- Follow the project's technical decisions

Thank you for contributing to Hector! 🚀
