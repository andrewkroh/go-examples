package client

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	// access-list access-list-number
	// { deny | permit } protocol
	// source source-wildcard
	// destination destination-wildcard
	// [ precedence precedence ] [ tos tos ] [ fragments ] [ log [ log-input ] | smartlog ] [ time-range time-range-name ] [ dscp dscp ]
	extendedAccessListRuleIPNet = `access-list 100 permit gre 192.168.1.0 0.0.0.255 10.3.0.0 0.0.255.255`

	// access-list access-list-number
	// { deny | permit } protocol
	// any
	// any
	// [ precedence precedence ] [ tos tos ] [ fragments ] [ log [log-input ] | smartlog] [ time-range time-range-name ] [ dscp dscp ]
	extendedAccessListRuleIPAny = `access-list 100 permit udp 192.168.1.0 0.0.0.255 any`

	// access-list access-list-number
	// { deny | permit } protocol
	// host source
	// host destination
	// [ precedence precedence ] [ tos tos ] [ fragments ] [ log [log-input ] | smartlog] [ time-range time-range-name ] [ dscp dscp ]
	extendedAccessListRuleIPHost = `access-list 100 deny   ip host 192.168.1.2 any log`

	// access-list access-list-number
	// { deny | permit } tcp
	// source source-wildcard [ operator port ]
	// destination destination-wildcard [ operator port ]
	// [ established ]
	// [ precedence precedence ] [ tos tos ] [ fragments ] [ log [log-input ] | smartlog] [ time-range time-range-name ] [ dscp dscp ]
	// [ flag ]
	extendedAccessListRuleExtendedTCP = `access-list 100 permit tcp 192.168.1.0 0.0.0.255 gt 1024 any eq http`

	// access-list access-list-number
	// { deny | permit } udp
	// source source-wildcard [ operator port ]
	// destination destination-wildcard [ operator port ]
	// [ precedence precedence ] [ tos tos ] [ fragments ] [ log [log-input ] | smartlog] [ time-range time-range-name ] [ dscp dscp ]
	extendedAccessListRuleExtendedUDP = `access-list 100 permit udp 192.168.1.0 0.0.0.255 eq 53 any gt 1024`

	// access-list access-list-number
	// { deny | permit } icmp
	// source source-wildcard
	// destination destination-wildcard
	// [ icmp-type | [[ icmp-type icmp-code ] | [ icmp-message ]]
	// [ precedence precedence ] [ tos tos ] [ fragments ] [ log [log-input ] | smartlog] [ time-range time-range-name ] [ dscp dscp ]
	extendedAccessListRuleExtendedICMP     = `access-list 100 permit icmp any any echo-reply`
	extendedAccessListRuleExtendedICMPCode = `access-list 100 permit icmp any any 3 1`

	//  access-list access-list-number
	// { deny | permit } igmp
	// source source-wildcard
	// destination destination-wildcard
	// [ igmp-type ]
	// [ precedence precedence ] [ tos tos ] [ fragments ] [ log [log-input ] | smartlog] [ time-range time-range-name ] [ dscp dscp ]
	extendedAccessListRuleExtendedIGMP = `access-list 100 permit igmp any any host-query`

	extendedAccessListRuleExtendedRemark = `access-list 100 remark Do not allow Smith through`
)

func TestParseExtendedAccessListIPv4(t *testing.T) {
	var testCases = []struct {
		name string
		line string
		rule AccessListEntry
	}{
		{
			name: "network",
			line: extendedAccessListRuleIPNet,
			rule: AccessListEntry{
				ID:                  "100",
				Permit:              true,
				Protocol:            "gre",
				Source:              "192.168.1.0",
				SourceWildcard:      "0.0.0.255",
				Destination:         "10.3.0.0",
				DestinationWildcard: "0.0.255.255",
			},
		},
		{
			name: "any",
			line: extendedAccessListRuleIPAny,
			rule: AccessListEntry{
				ID:             "100",
				Permit:         true,
				Protocol:       "udp",
				Source:         "192.168.1.0",
				SourceWildcard: "0.0.0.255",
				Destination:    "any",
			},
		},
		{
			name: "host",
			line: extendedAccessListRuleIPHost,
			rule: AccessListEntry{
				ID:          "100",
				Permit:      false,
				Protocol:    "ip",
				Source:      "192.168.1.2",
				Destination: "any",
				Log:         true,
			},
		},
		{
			name: "tcp",
			line: extendedAccessListRuleExtendedTCP,
			rule: AccessListEntry{
				ID:              "100",
				Permit:          true,
				Protocol:        "tcp",
				Source:          "192.168.1.0",
				SourceWildcard:  "0.0.0.255",
				SourcePort:      "gt 1024",
				Destination:     "any",
				DestinationPort: "eq http",
			},
		},
		{
			name: "udp",
			line: extendedAccessListRuleExtendedUDP,
			rule: AccessListEntry{
				ID:              "100",
				Permit:          true,
				Protocol:        "udp",
				Source:          "192.168.1.0",
				SourceWildcard:  "0.0.0.255",
				SourcePort:      "eq 53",
				Destination:     "any",
				DestinationPort: "gt 1024",
			},
		},
		{
			name: "icmp",
			line: extendedAccessListRuleExtendedICMP,
			rule: AccessListEntry{
				ID:          "100",
				Permit:      true,
				Protocol:    "icmp",
				Source:      "any",
				Destination: "any",
				ICMPType:    "echo-reply",
			},
		},
		{
			name: "icmp with code",
			line: extendedAccessListRuleExtendedICMPCode,
			rule: AccessListEntry{
				ID:          "100",
				Permit:      true,
				Protocol:    "icmp",
				Source:      "any",
				Destination: "any",
				ICMPType:    "3", // Destination unreachable
				ICMPCode:    "1", // Host is unreachable
			},
		},
		{
			name: "igmp",
			line: extendedAccessListRuleExtendedIGMP,
			rule: AccessListEntry{
				ID:          "100",
				Permit:      true,
				Protocol:    "igmp",
				Source:      "any",
				Destination: "any",
				IGMPType:    "host-query",
			},
		},
		{
			name: "remark",
			line: extendedAccessListRuleExtendedRemark,
			rule: AccessListEntry{
				ID:     "100",
				Remark: "Do not allow Smith through",
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Verify that the rule can be turned back into a string
			// that matches the original.
			t.Run("to string", func(t *testing.T) {
				assertEqualIgnoreWhitespace(t, tc.line, tc.rule.String())
			})

			// Verify the rule parsed correctly.
			t.Run("rule parsed", func(t *testing.T) {
				entry := &AccessListEntry{}
				if err := entry.Parse(tc.line); err != nil {
					t.Fatal(err)
				}

				assert.Equal(t, tc.rule, *entry)
			})
		})
	}
}
func TestExtendedAccessListRule_String(t *testing.T) {
	rule := AccessListEntry{
		ID:          "101",
		Permit:      true,
		Protocol:    "tcp",
		Source:      "any",
		Destination: "any",
		Established: true,
		Log:         true,
	}

	assert.Equal(t, `access-list 101 permit tcp any any established log`, rule.String())
}

func TestExtendedAccessList_String(t *testing.T) {
	accessList := AccessList{
		ID: "100",
		Rules: []AccessListEntry{
			{
				ID:     "100",
				Remark: "Deny RFC1918 access.",
			},
			{
				ID:                  "100",
				Permit:              false,
				Protocol:            "ip",
				Source:              "any",
				Destination:         "10.0.0.0",
				DestinationWildcard: "0.255.255.255",
				Log:                 true,
			},
		},
	}

	assert.Equal(t, `access-list 100 remark Deny RFC1918 access.
access-list 100 deny ip any 10.0.0.0 0.255.255.255 log
`, accessList.String())
}

func TestExtendedAccessListsFromString_Multi(t *testing.T) {
	const data = `access-list 100 remark ACL 100
access-list 199 remark ACL 199`

	list, err := ParseAccessListEntries(data)
	if err != nil {
		t.Fatal(err)
	}

	if assert.Len(t, list, 2) {
		assert.Equal(t, list[0].ID, "100")
		assert.Equal(t, list[1].ID, "199")
	}
}

const singleSpace = " "

var multiSpaceRe = regexp.MustCompile(`(?m)\s{2,}`)

func assertEqualIgnoreWhitespace(t testing.TB, expected, observed string) {
	t.Helper()
	expected = multiSpaceRe.ReplaceAllString(expected, singleSpace)
	observed = multiSpaceRe.ReplaceAllString(observed, singleSpace)
	assert.Equal(t, expected, observed)
}
