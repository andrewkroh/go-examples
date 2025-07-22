package programs

import (
	"context"
	"fmt"
	"os/exec"
)

func Execute(ctx context.Context, c *http.Client, callback func(event map[string]any)) error {
	cmd := exec.Command("ls", "/")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("subprocess failed: %v", err)
	}
	callback(map[string]any{"subprocess": string(output)})
	return nil
}
