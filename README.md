[English](README.md) | [한국어](README.ko.md)

# logsqueeze

[![CI](https://github.com/frontdoorrr/logsqueeze/actions/workflows/ci.yml/badge.svg)](https://github.com/frontdoorrr/logsqueeze/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/frontdoorrr/logsqueeze.svg)](https://pkg.go.dev/github.com/frontdoorrr/logsqueeze)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

Drop-in log compression for Claude Code and any MCP-compatible agent.

---

## Why

Claude Code's context fills up fast when you paste large log files. logsqueeze compresses logs locally using the XDrain algorithm — collapsing millions of lines into a handful of templates — before they ever reach the model. No data leaves your machine.

---

## How it looks

```
Compressed 1,200,000 lines → 12 templates (100,000x compression)

[x1,185,000] worker ready shard=<*> [1..48]
  samples: 14:22:11 worker ready shard=1 | 14:22:12 worker ready shard=2

[x12,000] pool acquire <*> [20..480ms p50=240ms]
  samples: 14:22:15 pool acquire 240ms | 14:22:16 pool acquire 480ms

[x3] ERROR psycopg2.OperationalError: connection <*> [timeout,refused,reset]
  samples: 14:22:16 ERROR psycopg2.OperationalError: connection timeout
```

Each `<*>` slot summarizes what varied in that position: numeric ranges with p50, or a list of distinct string values.

---

## Install

```bash
go install github.com/frontdoorrr/logsqueeze@latest
```

---

## Claude Code setup

Add to `~/.claude.json`:

```json
{
  "mcpServers": {
    "logsqueeze": {
      "command": "logsqueeze",
      "args": ["mcp", "serve"]
    }
  }
}
```

Restart Claude Code and the three tools below are available in every session.

---

## MCP Tools

| Tool | Description |
|------|-------------|
| `compress_logs` | Compress raw log text passed inline |
| `compress_file` | Read a log file by path and compress it |
| `compress_command` | Run a shell command and compress its stdout |

**Examples Claude Code can call:**

```
# Inline log text
compress_logs(logs="<paste log content>")

# Log file
compress_file(path="/var/log/app.log")

# Live command output
compress_command(command="kubectl logs -n prod deploy/api --tail=5000")
compress_command(command="docker logs --tail=2000 my-container")
compress_command(command="journalctl -u nginx --since '1 hour ago'")
```

---

## How it works

logsqueeze implements **XDrain**, a log template mining algorithm that groups lines by structural similarity using a prefix tree. Incoming lines are processed in shuffled batches (removing ordering bias), and each line is tried against multiple token rotations before a group is chosen by majority vote. Variable positions become `<*>` wildcards; numeric slots accumulate all seen values for accurate min/max/p50 statistics; string slots collect up to 6 distinct samples.

The result is a compact, human-readable summary that gives an LLM the statistical shape of the logs without the raw volume.

---

## Supported log formats

logsqueeze parses automatically — no `format` flag required in most cases.

| Format | Example |
|--------|---------|
| ISO 8601 syslog | `2024-01-01T14:22:11Z INFO worker ready` |
| Space-separated syslog | `2024-01-01 14:22:11 [INFO] worker ready` |
| Slash-date syslog | `2024/01/01 14:22:11 INFO worker ready` |
| JSON (structured) | `{"time":"...","level":"info","msg":"worker ready"}` |
| Level-only | `INFO: worker ready` / `[ERROR] pool acquire 240ms` |
| Plain text | `worker ready shard=1` |

JSON fields recognized: `time`/`timestamp`/`ts`/`@timestamp`, `level`/`severity`/`lvl`, `msg`/`message`/`log`/`text`.

---

## License

MIT — see [LICENSE](LICENSE).
