package programs

import (
	"context"
	"net/http"
	"os"
)

func Execute(ctx context.Context, c *http.Client, callback func(event map[string]any)) error {
	_ = os.Setenv("EVIL_VAR", "1234")
	val := os.Getenv("EVIL_VAR")
	callback(map[string]any{"environ": val})
	return nil
}
