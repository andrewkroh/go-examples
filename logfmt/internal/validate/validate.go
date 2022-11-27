package validate

type MessageTest struct {
	Msg      string
	Expected []string
	Error    string
}

var MessageTestCases = []MessageTest{
	{Msg: `key=""`, Expected: []string{"key", ""}},
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
