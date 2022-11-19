package logfmt

import (
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/alecthomas/participle/v2/lexer"
)

type MessageTest struct {
	Msg      string
	Expected []string
	Error    string
}

var messageTestCases = []MessageTest{
	//{Msg: `key=""`, Expected: []string{"key", ""}}, // TODO: Investigate panic.
	{Msg: `key="value"`, Expected: []string{"key", "value"}},
	{Msg: "key=value", Expected: []string{"key", "value"}},
	{Msg: "key= ", Expected: []string{"key", ""}},
	{Msg: `key="\""`, Expected: []string{"key", `"`}},
	{Msg: "key= key2=value", Expected: []string{"key", "", "key2", "value"}},
	{Msg: "key=/foobar", Expected: []string{"key", "/foobar"}},
	{Msg: "key=foo_bar", Expected: []string{"key", "foo_bar"}},
	{Msg: "key=foo@bar.com", Expected: []string{"key", "foo@bar.com"}},
	{Msg: "key=foobar^", Expected: []string{"key", "foobar^"}},
	{Msg: "key=+/-_^@f.oobar", Expected: []string{"key", "+/-_^@f.oobar"}},
	{Msg: `key="foo\n\rbar"`, Expected: []string{"key", "foo\n\rbar"}},
	{Msg: `key="foobar$"`, Expected: []string{"key", "foobar$"}},
	{Msg: `key="&foobar"`, Expected: []string{"key", "&foobar"}},
	{Msg: `key="x y"`, Expected: []string{"key", "x y"}},
	{Msg: `key="x,y"`, Expected: []string{"key", "x,y"}},
	{Msg: `key="value" key2="value2"`, Expected: []string{"key", "value", "key2", "value2"}},
	{Msg: "my_key= ", Expected: []string{"my_key", ""}},
	{Msg: "my.key= ", Expected: []string{"my.key", ""}},
	{Msg: "my%key= ", Expected: []string{"my%key", ""}},
	// From: https://www.brandur.org/logfmt
	{Msg: `key="undefined method ` + "`" + `serialize' for nil:NilClass"`, Expected: []string{"key", "undefined method `serialize' for nil:NilClass"}},
	// From: https://github.com/kr/logfmt/blob/19f9bcb100e6bcb308b5db29c682de01e9b3f2e6/decode.go#L5
	{
		Msg: `foo=bar a=14 baz="hello kitty" cool%story=bro f %^asdf`,
		Expected: []string{
			"foo", "bar",
			"a", "14",
			"baz", "hello kitty",
			"cool%story", "bro",
			"f", "",
			"%^asdf", "",
		},
		Error: "",
	},
}

func TestLogfmt(t *testing.T) {
	for i, tc := range messageTestCases {
		tc := tc
		t.Run(tc.Msg, func(t *testing.T) {
			exr, err := parser.ParseString("test_case_"+strconv.Itoa(i), tc.Msg)
			if err != nil {
				t.Error("ParseString failed with:", err)
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
	for _, tc := range messageTestCases {
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
