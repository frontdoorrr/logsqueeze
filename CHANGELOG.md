# Changelog

All notable changes to logsqueeze are documented here.

## [0.1.0] - 2026-06-08

### Added
- XDrain log template mining with batch shuffle and token rotation (multi-tree voting)
- Three MCP tools: `compress_logs`, `compress_file`, `compress_command`
- Numeric slot statistics: min, max, p50 computed across all observed values
- String slot sampling: up to 6 distinct values per wildcard position
- Automatic log format detection — no `--format` flag required
- Support for ISO 8601, space-separated, slash-date syslog, JSON (structured), level-only, and plain text formats
- JSON field aliases: `time`/`timestamp`/`ts`/`@timestamp`, `level`/`severity`/`lvl`, `msg`/`message`/`log`/`text`
- English and Korean README

[0.1.0]: https://github.com/frontdoorrr/logsqueeze/releases/tag/v0.1.0
