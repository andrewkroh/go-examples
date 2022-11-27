package participle

import (
	"testing"
)

func TestExpression(t *testing.T) {
	testCases := []struct {
		eval   string
		result bool
	}{
		{eval: "if var33", result: false},
		{eval: "if nil", result: false},
		{eval: "if 0", result: false},
		{eval: "if 0.0", result: false},
		{eval: "if true", result: true},
		{eval: "if false", result: false},
		{eval: `if "hello"`, result: true},

		{eval: "if var33 == varFalse", result: false},
		{eval: "if varFalse != nil", result: false},
		{eval: "if varNil == 0", result: false},
		{eval: "if var33 == 0.0", result: false},
		{eval: "if 1.1 == 11", result: false},
		{eval: "if varFalse == true", result: false},
		{eval: "if varFalse == false", result: true},
		{eval: `if varFalse != "hello"`, result: true},

		{eval: `if varFalse != "hello" == "world"`, result: false},
		{eval: `if varFalse != "hello" != "world"`, result: true},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.eval, func(t *testing.T) {
			program, err := parser.ParseString("", tc.eval)
			if err != nil {
				t.Fatal(err)
			}

			out := program.Eval(Context{
				vars: map[string]interface{}{
					"var33":    33,
					"varFalse": false,
					"varNil":   nil,
				},
			})
			t.Logf("[ %v ] -> %v", program.String(), out)

			if out != tc.result {
				t.Fatalf("got = %v, want = %v", out, tc.result)
			}
		})
	}
}

func BenchmarkExpressionEval(b *testing.B) {
	// 2022-11 on Apple M1 Max
	// BenchmarkExpressionEval-10  125676506  9.382 ns/op  0 B/op  0 allocs/op

	program, err := parser.ParseString("", `if myVar == "192.168.10.11"`)
	if err != nil {
		b.Fatal(err)
	}

	ctx := Context{
		map[string]interface{}{
			"myVar": "192.168.10.11",
		},
	}

	b.ResetTimer()
	var r interface{}
	for i := 0; i < b.N; i++ {
		r = program.Eval(ctx)
	}
	result = r.(bool)
}

var result bool

func BenchmarkIf(b *testing.B) {
	ctx := Context{
		map[string]interface{}{
			"myVar": "192.168.10.11",
		},
	}

	b.ResetTimer()
	var r bool
	for i := 0; i < b.N; i++ {
		r = nativeGoExpr(ctx)
	}
	result = r
}

func nativeGoExpr(ctx Context) bool {
	if val, found := ctx.vars["myVar"]; found {
		return val == "192.168.10.11"
	}
	return false
}
