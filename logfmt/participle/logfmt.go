// Package participle is a toy parser for log messages encoded in "logfmt"
// example message is:
//
//	at=info method=GET path=/ host=mutelight.org fwd="124.133.52.161"
//	dyno=web.2 connect=4ms service=8ms status=200 bytes=1653
//
// This is a toy because it's slow, and it can panic for some inputs. This
// was a learning experiment with github.com/alecthomas/participle/v2.
//
// References:
//
//	https://www.brandur.org/logfmt
//	https://github.com/sirupsen/logrus/blob/f8bf7650dccb756cea26edaf9217aab85500fe07/text_formatter.go
//
//nolint:govet,revive // Struct-tags use shorthand participle format for readability.
package participle

import (
	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

var (
	statefulLexer = lexer.MustStateful(lexer.Rules{
		"Root": {
			{"Ident", `[^ =]+`, lexer.Push("Punct")},
			{"whitespace", `[ \t]+`, nil},
		},
		"String": {
			{"Quoted", `"(\\"|[^"])*"`, lexer.Pop()},
			{"Unquoted", `[^" ]+`, lexer.Pop()},
			lexer.Return(),
		},
		"Punct": {
			{`Separator`, `=`, lexer.Push("String")},
			lexer.Return(),
		},
	})

	parser = participle.MustBuild[Message](
		participle.Lexer(statefulLexer),
		participle.Unquote("Quoted"),
	)
)

type Message struct {
	KeyValuePairs []Pair `@@*`
}

type Pair struct {
	Key   string `@Ident`
	Value string `( "=" @(Quoted | Unquoted)? )?`
}

// Parse parses a log message encoded in "logfmt".
func Parse(message string) (*Message, error) {
	return parser.ParseString("", message)
}
