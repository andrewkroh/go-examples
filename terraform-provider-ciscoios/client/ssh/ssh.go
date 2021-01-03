package ssh

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strings"

	"github.com/andrewkroh/go-examples/terraform-provider-ciscoios/client"
	"golang.org/x/crypto/ssh"
)

var _ client.Commander = (*Client)(nil)

type Client struct {
	address string
	config  *ssh.ClientConfig
	client  *ssh.Client
	session *ssh.Session
	shell   *shell
}

func NewClient(address, username, password string) (*Client, error) {
	if address == "" {
		return nil, errors.New("address is required")
	}
	if username == "" {
		return nil, errors.New("username is required")
	}
	if password == "" {
		return nil, errors.New("password is required")
	}

	// Append the default SSH port if missing.
	if _, _, err := net.SplitHostPort(address); err != nil {
		if addrErr, ok := err.(*net.AddrError); ok && strings.Contains(addrErr.Err, "missing port") {
			address = net.JoinHostPort(address, "22")
		}
	}

	return &Client{
		address: address,
		config:  newConfig(username, password),
	}, nil
}

func newConfig(username, password string) *ssh.ClientConfig {
	conf := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			log.Printf("Key for %v (%v): %v", hostname, remote.String(), hex.EncodeToString(key.Marshal()))
			return nil
		},
	}
	conf.SetDefaults()

	// Add older ciphers used by Cisco.
	conf.Ciphers = append(conf.Ciphers, "aes128-cbc", "3des-cbc", "aes192-cbc", "aes256-cbc")

	return conf
}

func (c *Client) connect() error {
	var err error

	// Connect to the remote server and perform the SSH handshake.
	c.client, err = ssh.Dial("tcp", c.address, c.config)
	if err != nil {
		return fmt.Errorf("failed to establish SSH connection: %w", err)
	}

	// Create a session
	c.session, err = c.client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %w", err)
	}

	c.shell, err = newShell(c.session)
	if err != nil {
		return fmt.Errorf("failed to create SSH shell: %w", err)
	}

	if _, err = c.shell.Command("terminal length 0"); err != nil {
		return fmt.Errorf("failed to set terminal length 0: %w", err)
	}

	return nil
}

func (c *Client) Close() error {
	if c.shell != nil {
		if _, err := c.shell.Command("logout"); err != nil && !errors.Is(err, io.EOF) {
			log.Fatal("failed while logging out", err)
		}
	}

	// TODO: Collect all errors.
	if c.client != nil {
		c.client.Close()
	}
	if c.session != nil {
		c.session.Close()
	}
	if c.shell != nil {
		c.shell.Close()
	}
	return nil
}

func (c *Client) Command(cmd string) ([]byte, error) {
	if c.shell == nil {
		if err := c.connect(); err != nil {
			return nil, err
		}
	}

	return c.shell.Command(cmd)
}
