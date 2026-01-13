# Contributing to hotbrew

Thanks for your interest in contributing! ☕

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/YOUR_USERNAME/hotbrew.git`
3. Install Go 1.21+
4. Run `make deps` to download dependencies
5. Run `make build` to build the binary
6. Run `./hotbrew` to test

## Development

```bash
# Build and run
make run

# Run tests
make test

# Format code
make fmt

# Lint (requires golangci-lint)
make lint
```

## Adding a New Source

Sources live in `internal/sources/`. To add a new source:

1. Create a new directory: `internal/sources/mysource/`
2. Implement the `source.Source` interface:

```go
type Source interface {
    Name() string
    Icon() string
    Fetch(ctx context.Context, cfg Config) (*Section, error)
    TTL() time.Duration
}
```

3. Register it in `internal/ui/app.go`
4. Add config options to `internal/config/config.go`

See `internal/sources/hackernews/` for a complete example.

## Adding a New Theme

Themes live in `internal/ui/theme/`. To add a new theme:

1. Create a new file: `internal/ui/theme/mytheme.go`
2. Implement the `Theme` interface
3. Register it in `internal/ui/theme/theme.go`

## Pull Request Process

1. Create a feature branch: `git checkout -b feature/my-feature`
2. Make your changes
3. Run `make fmt` and `make test`
4. Commit with a clear message
5. Push and open a PR

## Code Style

- Run `gofmt` before committing
- Keep functions small and focused
- Add comments for exported functions
- Follow existing patterns in the codebase

## Questions?

Open an issue or start a discussion. We're happy to help!
