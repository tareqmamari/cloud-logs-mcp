# Contributing to IBM Cloud Logs MCP Server

Thank you for your interest in contributing! This document provides guidelines for contributing to the project.

## Code of Conduct

This project follows the IBM Code of Conduct. By participating, you are expected to uphold this code.

## Getting Started

1. **Fork the repository**
2. **Clone your fork**:
   ```bash
   git clone https://github.com/your-username/logs-mcp-server.git
   cd logs-mcp-server
   ```

3. **Set up development environment**:
   ```bash
   make setup
   ```

4. **Create a branch**:
   ```bash
   git checkout -b feature/your-feature-name
   ```

## Development Workflow

### Building

```bash
make build
```

### Running Tests

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run only unit tests
make test-unit
```

### Code Quality

Before submitting, ensure your code passes all checks:

```bash
# Format code
make fmt

# Run linters
make lint

# Run security checks
make sec

# Check for vulnerabilities
make vuln

# Run all checks
make check
```

## Coding Standards

### Go Style Guide

- Follow the [Effective Go](https://golang.org/doc/effective_go.html) guidelines
- Use `gofmt` for formatting (run `make fmt`)
- Write clear, self-documenting code
- Add comments for exported functions and types
- Keep functions small and focused

### Error Handling

```go
// ✅ Good - Return errors, don't panic
func DoSomething() error {
    if err := validate(); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }
    return nil
}

// ❌ Bad - Don't panic in library code
func DoSomething() {
    if err := validate(); err != nil {
        panic(err)
    }
}
```

### Logging

Use structured logging with zap:

```go
logger.Info("Operation completed",
    zap.String("operation", "create_alert"),
    zap.Duration("duration", elapsed),
    zap.Int("status_code", 200),
)
```

### Testing

- Write tests for all new functionality
- Aim for >80% code coverage
- Use table-driven tests where appropriate
- Mock external dependencies

Example:

```go
func TestSomething(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"valid input", "test", "result", false},
        {"invalid input", "", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Function(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
            }
            if got != tt.want {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

## Pull Request Process

1. **Update documentation** if you're changing functionality
2. **Add tests** for new features
3. **Run all checks**: `make check`
4. **Update CHANGELOG.md** with your changes
5. **Commit with clear messages**:
   ```
   feat: add support for custom queries

   - Implemented custom query parser
   - Added validation for query syntax
   - Updated documentation

   Fixes #123
   ```

6. **Push to your fork** and create a pull request

### Commit Message Format

We follow [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` - New feature
- `fix:` - Bug fix
- `docs:` - Documentation changes
- `test:` - Adding tests
- `refactor:` - Code refactoring
- `perf:` - Performance improvements
- `chore:` - Maintenance tasks

### Pull Request Template

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
- [ ] Unit tests added/updated
- [ ] Integration tests added/updated
- [ ] Manual testing performed

## Checklist
- [ ] Code follows style guidelines
- [ ] Self-review completed
- [ ] Documentation updated
- [ ] Tests pass locally
- [ ] No new warnings
```

## Security

- **Never commit secrets** (API keys, passwords, etc.)
- Report security vulnerabilities to security@ibm.com
- Follow secure coding practices
- Use parameterized queries to prevent injection

## Adding New Tools

To add a new MCP tool:

1. Create tool implementation in `internal/tools/`:
   ```go
   type NewTool struct {
       *BaseTool
   }

   func NewNewTool(client *client.Client, logger *zap.Logger) *NewTool {
       return &NewTool{BaseTool: NewBaseTool(client, logger)}
   }

   func (t *NewTool) Name() string {
       return "new_tool_name"
   }

   func (t *NewTool) Description() string {
       return "Description of what the tool does"
   }

   func (t *NewTool) InputSchema() mcp.ToolInputSchema {
       // Define input parameters
   }

   func (t *NewTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
       // Implement tool logic
   }
   ```

2. Register tool in `internal/server/server.go`:
   ```go
   s.registerTool(tools.NewNewTool(s.apiClient, s.logger))
   ```

3. Add tests in `internal/tools/new_tool_test.go`

4. Update README.md with new tool documentation

## Documentation

- Keep README.md up to date
- Document all exported functions and types
- Add examples for complex functionality
- Update API documentation when changing interfaces

## Questions?

- Open an issue for questions
- Check existing issues and PRs first
- Be respectful and constructive

Thank you for contributing! 🎉
