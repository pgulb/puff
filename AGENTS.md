# Agent Instructions for puff

This file contains instructions for AI coding agents working on the puff project. Puff is a simple binary package manager for downloading and updating binary releases from GitHub repositories.

## Project Overview
- **Language**: Go 1.25.3
- **Purpose**: Package manager for GitHub binary releases
- **Architecture**: CLI tool with configuration stored in `~/.config/puff/`

## Featured Repositories Management

### Adding New Featured Repositories
- when adding, make sure to do steps from 'Testing New Repositories' section
- Featured repositories are predefined popular CLI tools that users can install with `puff add <repo>`
- Located in `metadata.go` in the `AvailableRepos()` function
- Each repo requires:
  - `Path`: GitHub repository path (e.g., "sharkdp/bat")
  - `Desc`: Short description of the tool
  - `Regexp`: Regex pattern to match the Linux x86_64 binary asset in GitHub releases

### Repository Selection Criteria
- Must provide pre-compiled static Linux x86_64 binaries in GitHub releases (actual binary executables, not Python wheels or other non-binary formats)
- Popular and widely-used CLI tools for development/DevOps
- No complex dependencies or installation requirements
- Not GUI applications or tools requiring special terminal features
- Complements existing featured repos without significant overlap

### Testing New Repositories
1. Verify GitHub releases contain Linux x86_64 binaries
2. Test regex pattern matches the correct asset name
3. Build the project using `task build`
4. Run `puff add <repo>` with the newly built binary to install the repo
5. Using the absolute path of the installed binary, run it with `--help` to confirm it was downloaded and displays help
6. DO NOT update this document with new added repo names

### Rejected Repositories
See `rejected_repos.txt` for repositories that were considered but rejected for inclusion. This file contains only repo paths (one per line) to avoid re-evaluating the same repos. Do not add descriptions or reasons to this file.

## Build/Lint/Test Commands

### Building
```bash
# Build the binary
task build
```

### Linting and Formatting
```bash

# Vet for suspicious code
go1.25.3 vet ./...
```

## Code Style Guidelines

### Imports
- Group imports: standard library first, then blank line, then third-party packages
- Use parentheses for multi-line imports
- Remove unused imports automatically

```go
import (
    "fmt"
    "os"
    "path/filepath"

    "github.com/some/package"
)
```

### Naming Conventions
- **Functions/Methods**: PascalCase for exported, camelCase for unexported
- **Variables**: camelCase throughout
- **Constants**: PascalCase for exported, camelCase for unexported
- **Types/Structs**: PascalCase
- **Package name**: lowercase, single word (current: `puff`)

### Error Handling
- Check errors immediately after function calls
- Use `if err != nil` pattern consistently
- Return errors up the call stack rather than handling globally
- Use `fmt.Errorf` for wrapping errors with context

```go
func example() error {
    result, err := someFunction()
    if err != nil {
        return fmt.Errorf("failed to get result: %w", err)
    }
    return nil
}
```

### Function Structure
- Public functions start with capital letters
- Add comments above public functions explaining their purpose
- Keep functions focused on single responsibilities
- Use early returns to reduce nesting

```go
// exampleFunction processes the input and returns a result
func exampleFunction(input string) (string, error) {
    if input == "" {
        return "", errors.New("input cannot be empty")
    }

    // Process input
    result := processInput(input)
    return result, nil
}
```

### File Organization
- Main entry point: `cmd/main.go`
- Library code: root level `.go` files
- Configuration/setup: `setup.go`
- Binary operations: `bins.go`
- Metadata handling: `metadata.go`
- API interactions: `gh_api.go`

### Comments
- Add package comments for non-main packages
- Document exported functions with clear descriptions
- Use `//` for single-line comments
- Keep comments concise but informative

### Constants and Types
- Define constants for magic numbers/strings
- Use meaningful names for custom types
- Group related constants together

```go
const (
    DefaultTimeout = 30
    ConfigDir      = "puff"
)

type Config struct {
    Timeout int
    Path    string
}
```

### Path Handling
- Use `filepath.Join` instead of string concatenation
- Use `os.UserConfigDir()` and `os.UserHomeDir()` for user directories
- Validate paths before using them

### Security Considerations
- Store sensitive data (like GitHub PAT) with appropriate file permissions (0600)
- Validate user input before processing
- Use HTTPS URLs by default
- Avoid logging sensitive information

### Testing Guidelines
- Create `*_test.go` files alongside code being tested
- Use table-driven tests for multiple test cases
- Test error conditions explicitly
- Use `t.Run` for subtests to organize test suites

```go
func TestExample(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {"valid input", "test", "TEST", false},
        {"empty input", "", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := exampleFunction(tt.input)
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.expected, result)
            }
        })
    }
}
```

### Dependencies
- Keep dependencies minimal
- Use Go standard library when possible
- Document any new dependencies in PR descriptions
- Update `go.mod` appropriately

### Configuration Management
- Store config in `~/.config/puff/`
- Use JSON for structured data storage
- Create directories with appropriate permissions (0750)
- Handle missing config files gracefully

### CLI Interface
- Use `os.Args` for argument parsing (keep simple)
- Provide clear usage messages
- Exit with code 1 for errors, 0 for success
- Use consistent output formatting

### Version Management
- Keep version as a constant in code
- Update version for releases
- Use semantic versioning

### Git Workflow
- Commit messages should be clear and descriptive
- Use conventional commits when possible: `feat:`, `fix:`, `docs:`, etc.
- Keep commits focused on single changes
- Test before committing

### Performance Considerations
- Avoid unnecessary allocations in hot paths
- Use buffered I/O for file operations
- Consider memory usage for large downloads
- Profile performance-critical code

### Logging
- Use `fmt.Printf` for user-facing output
- Log errors to stderr
- Consider adding structured logging for debugging
- Don't log sensitive information

### Platform Compatibility
- Test on Linux, macOS, and Windows when possible
- Use `runtime.GOOS` and `runtime.GOARCH` for platform-specific code
- Handle path separators correctly with `filepath`

This document should be updated as the codebase evolves and new patterns emerge.</content>
<parameter name="filePath">AGENTS.md