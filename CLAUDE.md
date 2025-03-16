# KarpelesLab/rest Development Guide

## Build & Test Commands
- Build: `make all` (runs goimports and builds package)
- Dependencies: `make deps` (gets dependencies)
- Run all tests: `make test` or `go test -v ./...`
- Run single test: `go test -v github.com/KarpelesLab/rest -run TestUpload`

## Code Style Guidelines
- **Imports**: Use goimports (enforced in Makefile)
- **Formatting**: Standard Go format
- **Go Version**: Go 1.21 (toolchain 1.21.3)

## Naming Conventions
- PascalCase for exported identifiers
- camelCase for unexported identifiers
- Error variables prefixed with "Err" (e.g., ErrLoginRequired)

## Error Handling
- Use custom error types with Unwrap() implementation
- Context-based error propagation
- HTTP errors mapped to standard errors (e.g., os.ErrPermission)
- Use errors.New and fmt.Errorf for error creation

## Types & Structure
- Use generics (Go 1.21 feature)
- Context-based configuration
- Single package with well-defined responsibilities
- CLI tools in separate directories