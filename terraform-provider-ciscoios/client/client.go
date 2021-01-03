package client

import (
	"fmt"
	"sort"
)

type Commander interface {
	Command(string) ([]byte, error)

	Close() error
}

type Client struct {
	commander Commander
}

func New(commander Commander) (*Client, error) {
	return &Client{commander: commander}, nil
}

func (c *Client) Close() error {
	return c.commander.Close()
}

func (c *Client) ACLs() ([]AccessList, error) {
	conf, err := c.commander.Command("show running-config | include access-list")
	if err != nil {
		return nil, err
	}

	entries, err := ParseAccessListEntries(string(conf))
	if err != nil {
		return nil, err
	}

	groups := map[string][]AccessListEntry{}
	for _, e := range entries {
		list, found := groups[e.ID]
		if !found {
			groups[e.ID] = []AccessListEntry{e}
			continue
		}
		list = append(list, e)
		groups[e.ID] = list
	}

	out := make([]AccessList, 0, len(groups))
	for id, accessList := range groups {
		out = append(out, AccessList{ID: id, Rules: accessList})
	}

	// Sort all rules by ID.
	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})

	return out, nil
}

func (c *Client) DeleteACL(id string) error {
	if _, err := c.commander.Command("configure terminal"); err != nil {
		return err
	}

	if _, err := c.commander.Command("no access-list " + id); err != nil {
		return err
	}

	if _, err := c.commander.Command("end"); err != nil {
		return err
	}
	return nil
}

func (c *Client) ACL(id string) (*AccessList, error) {
	acls, err := c.ACLs()
	if err != nil {
		return nil, err
	}

	for _, acl := range acls {
		if acl.ID == id {
			return &acl, nil
		}
	}

	return nil, fmt.Errorf("access-list %v not found", id)
}

func (c *Client) CreateACL(acl AccessList) error {
	_, err := c.commander.Command("configure terminal")
	if err != nil {
		return fmt.Errorf("failed to enter configure mode: %w", err)
	}

	if _, err = c.commander.Command(acl.String()); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	if _, err = c.commander.Command("end"); err != nil {
		return err
	}
	return nil
}
