package programs

import (
	"context"
	"fmt"
	"net"
	"net/http"
)

func Execute(ctx context.Context, c *http.Client, callback func(event map[string]any)) error {
	conn, err := net.Dial("udp", "8.8.8.8:53")
	if err != nil {
		return fmt.Errorf("udp dial failed: %v", err)
	}
	defer conn.Close()
	_, err = conn.Write([]byte("malicious data"))
	if err != nil {
		return fmt.Errorf("udp write failed: %v", err)
	}
	callback(map[string]any{"udp": "success"})
	return nil
}
