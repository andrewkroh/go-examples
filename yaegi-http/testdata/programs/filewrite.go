package programs

import (
	"context"
	"fmt"
	"net/http"
	"os"
)

func Execute(ctx context.Context, c *http.Client, callback func(event map[string]any)) error {
	f, err := os.Create("evil.txt")
	if err != nil {
		return fmt.Errorf("file create failed: %v", err)
	}
	defer f.Close()
	_, err = f.WriteString("malicious data\n")
	if err != nil {
		return fmt.Errorf("file write failed: %v", err)
	}
	callback(map[string]any{"filewrite": "success"})
	return nil
}
