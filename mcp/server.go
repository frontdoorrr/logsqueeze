package mcp

import (
	"context"
	"fmt"
	"strings"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/frontdoorrr/logsqueeze/drain"
	"github.com/frontdoorrr/logsqueeze/parser"
	"github.com/frontdoorrr/logsqueeze/source"
)

// Serve starts the MCP stdio server and blocks until stdin closes.
func Serve() error {
	s := server.NewMCPServer(
		"logsqueeze",
		"0.1.0",
		server.WithToolCapabilities(false),
	)

	s.AddTool(compressLogsTool(), handleCompressLogs)
	s.AddTool(compressFileTool(), handleCompressFile)
	s.AddTool(compressCommandTool(), handleCompressCommand)

	return server.ServeStdio(s)
}

// --- tool definitions ---

func compressLogsTool() mcpgo.Tool {
	return mcpgo.NewTool(
		"compress_logs",
		mcpgo.WithDescription("Compress log text into compact template groups for efficient LLM analysis. Use when log content is too large to fit in context."),
		mcpgo.WithString("logs", mcpgo.Required(), mcpgo.Description("Raw log text to compress")),
		mcpgo.WithString("format", mcpgo.Description("Log format hint: 'text' or 'json' (default: auto-detect)")),
	)
}

func compressFileTool() mcpgo.Tool {
	return mcpgo.NewTool(
		"compress_file",
		mcpgo.WithDescription("Read a log file and compress it into template groups."),
		mcpgo.WithString("path", mcpgo.Required(), mcpgo.Description("Absolute path to the log file, or '-' for stdin")),
		mcpgo.WithString("format", mcpgo.Description("Log format hint: 'text' or 'json' (default: auto-detect)")),
	)
}

func compressCommandTool() mcpgo.Tool {
	return mcpgo.NewTool(
		"compress_command",
		mcpgo.WithDescription("Run a shell command and compress its log output. Use for kubectl logs, docker logs, gcloud logging read, aws logs tail, etc."),
		mcpgo.WithString("command", mcpgo.Required(), mcpgo.Description("Shell command to run (executed via sh -c)")),
		mcpgo.WithString("format", mcpgo.Description("Log format hint: 'text' or 'json' (default: auto-detect)")),
	)
}

// --- handlers ---

func handleCompressLogs(_ context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	logsVal, err := req.RequireString("logs")
	if err != nil {
		return mcpgo.NewToolResultError(err.Error()), nil
	}
	lines := strings.Split(logsVal, "\n")
	return compress(lines), nil
}

func handleCompressFile(_ context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	path, err := req.RequireString("path")
	if err != nil {
		return mcpgo.NewToolResultError(err.Error()), nil
	}
	lines, err := source.ReadFile(path)
	if err != nil {
		return mcpgo.NewToolResultError(fmt.Sprintf("read file: %v", err)), nil
	}
	return compress(lines), nil
}

func handleCompressCommand(_ context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	command, err := req.RequireString("command")
	if err != nil {
		return mcpgo.NewToolResultError(err.Error()), nil
	}
	lines, err := source.RunCommand(command)
	if err != nil {
		return mcpgo.NewToolResultError(fmt.Sprintf("run command: %v", err)), nil
	}
	return compress(lines), nil
}

func compress(rawLines []string) *mcpgo.CallToolResult {
	logLines := parser.ParseAll(rawLines)
	if len(logLines) == 0 {
		return mcpgo.NewToolResultText("No log lines to compress.")
	}
	cfg := drain.DefaultConfig()
	result := drain.Analyze(logLines, cfg)
	return mcpgo.NewToolResultText(drain.Render(result))
}
