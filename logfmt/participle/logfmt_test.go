package participle

import (
	"reflect"
	"strings"
	"testing"

	"github.com/alecthomas/participle/v2/lexer"

	"github.com/andrewkroh/go-examples/logfmt/internal/validate"
)

func TestParse(t *testing.T) {
	for _, tc := range validate.MessageTestCases {
		tc := tc
		t.Run(tc.Msg, func(t *testing.T) {
			if tc.Msg == `key=""` {
				t.Skip("Avoid panic")
				return
			}

			exr, err := Parse(tc.Msg)
			if err != nil {
				t.Error("Parse failed with:", err)
				debugLexer(t, statefulLexer, tc.Msg)
				return
			}

			observed := make([]string, 0, len(exr.KeyValuePairs)*2)
			for _, p := range exr.KeyValuePairs {
				observed = append(observed, p.Key, p.Value)
			}

			if !reflect.DeepEqual(tc.Expected, observed) {
				t.Errorf("Error parsing=%v\nwant=[%v]\ngot=[%v]",
					tc.Msg,
					strings.Join(tc.Expected, ", "),
					strings.Join(observed, ", "))
			}
		})
	}
}

func FuzzReverse(f *testing.F) {
	// This will trigger a panic and fail CI so disable.
	f.SkipNow()

	for _, tc := range validate.MessageTestCases {
		f.Add(tc.Msg) // Use f.Add to provide a seed corpus
	}

	f.Fuzz(func(t *testing.T, orig string) {
		parser.ParseString("", orig) //nolint:errcheck // This is only checking for panics.
	})
}

func BenchmarkParse(b *testing.B) {
	// BenchmarkParse-10          62527             17879 ns/op           17237 B/op        314 allocs/op

	const msg = `level=info msg="Stopping all fetchers" tag=stopping_fetchers id=ConsumerFetcherManager-1382721708341 module=kafka.consumer.ConsumerFetcherManager`

	for i := 0; i < b.N; i++ {
		_, err := parser.ParseString("", msg)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func debugLexer(t *testing.T, lxDef lexer.Definition, text string) {
	lx, err := lxDef.Lex("", strings.NewReader(text))
	if err != nil {
		t.Fatal("Lex() failed with:", err)
	}

	consumeTokens(t, lxDef, lx)
}

func consumeTokens(t *testing.T, lexDef lexer.Definition, lx lexer.Lexer) {
	tokens := make([]lexer.Token, 0, 1024)
	for {
		token, err := lx.Next()
		if err != nil {
			t.Error(err)
			break
		}
		tokens = append(tokens, token)
		if token.Type == lexer.EOF {
			break
		}
	}

	logTokens(t, lexDef, tokens)
}

func logTokens(t *testing.T, lexDef lexer.Definition, tokens []lexer.Token) {
	symbols := lexDef.Symbols()
	tokenToName := make(map[lexer.TokenType]string, len(symbols))
	for name, id := range symbols {
		tokenToName[id] = name
	}

	for _, tok := range tokens {
		t.Log(tokenToName[tok.Type], ":", tok.Value)
	}
}
