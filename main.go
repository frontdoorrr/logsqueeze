package main

import (
	"fmt"
	"os"

	"github.com/frontdoorrr/logsqueeze/mcp"
)

const version = "0.1.0"

const usage = `logsqueeze - log compression MCP server

Usage:
  logsqueeze mcp serve       Start MCP stdio server
  logsqueeze version         Print version
  logsqueeze --version, -v   Print version

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
	if len(os.Args) >= 2 {
		switch os.Args[1] {
		case "mcp":
			if len(os.Args) >= 3 && os.Args[2] == "serve" {
				if err := mcp.Serve(); err != nil {
					fmt.Fprintf(os.Stderr, "logsqueeze: %v\n", err)
					os.Exit(1)
				}
				return
			}
		case "version", "--version", "-v":
			fmt.Println("logsqueeze v" + version)
			return
		}
	}
	fmt.Print(usage)
}
