package main

import (
	"fmt"
	"os"

	"github.com/frontdoorrr/logsqueeze/mcp"
)

const usage = `logsqueeze - log compression MCP server

Usage:
  logsqueeze mcp serve   Start MCP stdio server

Add to ~/.claude.json:
  {
    "mcpServers": {
      "logsqueeze": {
        "command": "logsqueeze",
        "args": ["mcp", "serve"]
      }
    }
  }
`

func main() {
	if len(os.Args) >= 3 && os.Args[1] == "mcp" && os.Args[2] == "serve" {
		if err := mcp.Serve(); err != nil {
			fmt.Fprintf(os.Stderr, "logsqueeze: %v\n", err)
			os.Exit(1)
		}
		return
	}
	fmt.Print(usage)
}
