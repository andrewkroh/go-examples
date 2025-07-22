package programs

import (
	"context"
	"fmt"
	"net"
	"net/http"
)

func Execute(ctx context.Context, c *http.Client, callback func(event map[string]any)) error {
	// Attempt to open a raw socket (should be blocked)
	conn, err := net.Dial("tcp", "198.51.100.1:22")
	if err != nil {
		return fmt.Errorf("raw socket dial failed: %v", err)
	}
	defer conn.Close()
	callback(map[string]any{"rawsocket": "success"})
	return nil
}
