package programs

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

func Execute(ctx context.Context, c *http.Client, callback func(event map[string]any)) error {
	flag, err := ioutil.ReadFile("flag.txt")
	if err != nil {
		return fmt.Errorf("could not read flag.txt: %v", err)
	}
	callback(map[string]any{
		"flag": string(flag),
	})

	if err := os.Remove("flag.txt"); err != nil {
		return fmt.Errorf("could not remove flag file: %v", err)
	}
	return nil
}
