package source

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const shellTimeout = 30 * time.Second

// RunCommand executes a shell command and returns its stdout lines.
func RunCommand(command string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), shellTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("command timed out after %s", shellTimeout)
		}
		// non-zero exit is fine — we still use whatever was printed
		if stdout.Len() == 0 {
			return nil, fmt.Errorf("command failed: %w", err)
		}
	}

	raw := stdout.String()
	lines := strings.Split(raw, "\n")
	// trim trailing empty line from the split
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines, nil
}
