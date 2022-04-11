package changelog

import (
	"fmt"
	"strings"
)

type ChangeType uint8

const (
	Bugfix ChangeType = iota + 1
	Enhancement
	BreakingChange
)

var changeTypeNames = map[ChangeType]string{
	Bugfix:         "bugfix",
	Enhancement:    "enhancement",
	BreakingChange: "breaking-change",
}

func (ct ChangeType) String() string {
	if name, found := changeTypeNames[ct]; found {
		return name
	}
	return "unknown"
}

func NewChangeType(s string) (ChangeType, error) {
	ct := strings.ToLower(s)

	switch {
	case strings.HasPrefix(ct, "bu") && strings.HasPrefix("bugfix", ct):
		return Bugfix, nil
	case strings.HasPrefix(ct, "e") && strings.HasPrefix("enhancement", ct):
		return Enhancement, nil
	case strings.HasPrefix(ct, "br") && strings.HasPrefix("breaking-change", ct):
		return BreakingChange, nil
	default:
		return 0, fmt.Errorf("invalid change type %q", ct)
	}
}
