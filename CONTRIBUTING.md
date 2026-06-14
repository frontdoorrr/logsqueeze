# Contributing to logsqueeze

## Prerequisites

- Go 1.21+
- `golangci-lint` (optional, for local lint: `brew install golangci-lint`)
- `gh` CLI (optional, for creating releases)

## Build and test

```bash
go build ./...
go vet ./...
go test -race ./...
```

## Run the MCP server locally

```bash
go run . mcp serve
```

Point Claude Code at the local build in `~/.claude.json`:

```json
{
  "mcpServers": {
    "logsqueeze": {
      "command": "go",
      "args": ["run", "/absolute/path/to/logsqueeze", "mcp", "serve"]
    }
  }
}
```

## Pull requests

- Open an issue first for significant changes.
- `gofmt` is enforced — the CI will reject unformatted code.
- Add tests for new behavior in `drain/drain_test.go` or `parser/parser_test.go`.
- Keep PRs focused: one feature or fix per PR.

## Adding a log format

New format parsers belong in `parser/parser.go`. Add a test case in `parser/parser_test.go` that covers the new format's timestamp, level, and message extraction.

## Algorithm changes

Core XDrain logic is in `drain/drain.go`. If you touch the algorithm, verify compression behavior hasn't regressed:

```bash
go test -race -count=3 ./drain/...
```
