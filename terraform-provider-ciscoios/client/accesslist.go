package client

import (
	"bufio"
	"bytes"
	"strings"
)

type AccessList struct {
	ID    string
	Rules []AccessListEntry
}

func (al AccessList) String() string {
	var sb strings.Builder
	for _, entry := range al.Rules {
		entry.ID = al.ID
		sb.WriteString(entry.String())
		sb.WriteByte('\n')
	}
	return sb.String()
}

type AccessListEntry struct {
	ID string

	Remark string

	Permit              bool   // permit / deny
	Protocol            string // 0 - 255 or name
	Source              string
	SourceWildcard      string
	SourcePort          string
	Destination         string
	DestinationWildcard string
	DestinationPort     string
	Established         bool
	ICMPType            string
	ICMPCode            string
	IGMPType            string
	Log                 bool
}

func (ale AccessListEntry) String() string {
	tokens := []string{"access-list", ale.ID}
	if ale.Remark != "" {
		tokens = append(tokens, "remark", ale.Remark)
		return strings.Join(tokens, " ")
	}

	if ale.Permit {
		tokens = append(tokens, "permit")
	} else {
		tokens = append(tokens, "deny")
	}

	tokens = append(tokens, ale.Protocol)

	if ale.Source != "" {
		if ale.Source == "any" || ale.SourceWildcard != "" {
			tokens = append(tokens, ale.Source)
		} else {
			tokens = append(tokens, "host "+ale.Source)
		}
	}
	if ale.SourceWildcard != "" {
		tokens = append(tokens, ale.SourceWildcard)
	}
	if ale.SourcePort != "" {
		tokens = append(tokens, ale.SourcePort)
	}

	if ale.Destination != "" {
		if ale.Destination == "any" || ale.DestinationWildcard != "" {
			tokens = append(tokens, ale.Destination)
		} else {
			tokens = append(tokens, "host "+ale.Destination)
		}
	}
	if ale.DestinationWildcard != "" {
		tokens = append(tokens, ale.DestinationWildcard)
	}
	if ale.DestinationPort != "" {
		tokens = append(tokens, ale.DestinationPort)
	}

	switch ale.Protocol {
	case "icmp":
		if ale.ICMPType != "" {
			tokens = append(tokens, ale.ICMPType)
		}
		if ale.ICMPCode != "" {
			tokens = append(tokens, ale.ICMPCode)
		}
	case "igmp":
		if ale.IGMPType != "" {
			tokens = append(tokens, ale.IGMPType)
		}
	case "tcp":
		if ale.Established {
			tokens = append(tokens, "established")
		}
	}

	if ale.Log {
		tokens = append(tokens, "log")
	}

	return strings.Join(tokens, " ")
}

func ParseAccessListEntries(data string) ([]AccessListEntry, error) {
	var out []AccessListEntry

	s := bufio.NewScanner(bytes.NewBufferString(data))
	for s.Scan() {
		line := s.Text()

		if !strings.HasPrefix(line, "access-list ") {
			continue
		}

		entry := &AccessListEntry{}
		if err := entry.Parse(line); err != nil {
			return nil, err
		}
		out = append(out, *entry)
	}

	if err := s.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
