package programs

import (
	"context"
	"fmt"
	"net"
	"net/http"
)

func Execute(ctx context.Context, c *http.Client, callback func(event map[string]any)) error {
	ln, err := net.Listen("tcp", ":9999")
	if err != nil {
		return fmt.Errorf("bind failed: %v", err)
	}
	defer ln.Close()
	callback(map[string]any{"bind": "success"})
	return nil
}
