// Package browser provides cross-platform browser launching.
package browser

import (
	"context"
	"os/exec"
	"runtime"
	"time"
)

// Open launches the given URL in the default browser. It times out
// after 10 seconds to avoid blocking the caller indefinitely.
func Open(ctx context.Context, url string) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.CommandContext(ctx, "open", url) //nolint:gosec // URL validated by caller
	case "windows":
		cmd = exec.CommandContext(ctx, "rundll32", "url.dll,FileProtocolHandler", url) //nolint:gosec // URL validated by caller
	default:
		cmd = exec.CommandContext(ctx, "xdg-open", url) //nolint:gosec // URL validated by caller
	}

	return cmd.Run()
}
