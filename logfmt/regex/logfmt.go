package regex

import (
	"regexp"
	"strconv"
)

const (
	key     = `(?P<key>[^= ]+)`
	unquote = `(?P<unquote>[^ "]+)`
	quote   = `(?P<quote>"(?:\\"|[^"])*")`
	value   = `(?:` + quote + `|` + unquote + `)?`
	equals  = `=`
	pair    = key + `(?:` + equals + value + `)?`
	msg     = pair
)

var (
	re         = regexp.MustCompile(msg)
	groupNames = re.SubexpNames()
)

type Message struct {
	KeyValuePairs []Pair
}

type Pair struct {
	Key   string
	Value string
}

// Parse parses a log message encoded in "logfmt".
func Parse(message string) (*Message, error) {
	var pairs []Pair

	for _, match := range re.FindAllStringSubmatch(message, -1) {
		var pair Pair
		for groupIdx, group := range match {
			name := groupNames[groupIdx]

			switch name {
			case "key":
				if group != "" {
					pair.Key = group
				}
			case "quote":
				if group != "" {
					var err error
					if pair.Value, err = strconv.Unquote(group); err != nil {
						return nil, err
					}
				}
			case "unquote":
				if group != "" {
					pair.Value = group
				}
			}
		}

		if pair.Key != "" {
			pairs = append(pairs, pair)
		}
	}

	return &Message{KeyValuePairs: pairs}, nil
}
