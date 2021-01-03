package ciscoios

import (
	"strings"

	"github.com/andrewkroh/go-examples/terraform-provider-ciscoios/client"
	"github.com/andrewkroh/go-examples/terraform-provider-ciscoios/client/file"
	"github.com/andrewkroh/go-examples/terraform-provider-ciscoios/client/ssh"
)

// Config is the configuration structure used to instantiate a
// new Cisco IOS client.
type Config struct {
	Address  string
	Username string
	Password string
}

// NewClient returns a new CiscoIOS client.
func (c *Config) NewClient() (*client.Client, error) {
	var cmdr client.Commander
	var err error
	if strings.HasPrefix(c.Address, "file://") {
		cmdr, err = file.NewClient(c.Address)
	} else {
		cmdr, err = ssh.NewClient(c.Address, c.Username, c.Password)
	}
	if err != nil {
		return nil, err
	}

	return client.New(cmdr)
}
