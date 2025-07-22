package programs

import (
	"context"
	"fmt"
	"net/http"
	"os"
)

func Execute(ctx context.Context, c *http.Client, callback func(event map[string]any)) error {
	err := os.Symlink("/etc/passwd", "passwd-link")
	if err != nil {
		return fmt.Errorf("symlink failed: %v", err)
	}
	callback(map[string]any{"symlink": "success"})
	return nil
}
