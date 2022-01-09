package file

import (
	"fmt"
	"io/ioutil"
	"net/url"

	"github.com/andrewkroh/go-examples/terraform-provider-ciscoios/client"
)

var _ client.Commander = (*Client)(nil)

type Client struct {
	runningConfig string
}

func NewClient(address string) (*Client, error) {
	fileURL, err := url.Parse(address)
	if err != nil {
		return nil, err
	}
	if fileURL.Scheme != "file" {
		return nil, fmt.Errorf("url must have a file scheme but had %q", fileURL.Scheme)
	}

	contents, err := ioutil.ReadFile(fileURL.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", fileURL.Path, err)
	}

	return &Client{
		runningConfig: string(contents),
	}, nil
}

func (c *Client) Command(cmd string) ([]byte, error) {
	return []byte(c.runningConfig), nil
}

func (c *Client) Close() error {
	return nil
}
