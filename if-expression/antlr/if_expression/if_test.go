package if_expression

import (
	"strings"
	"testing"
)

func TestIf(t *testing.T) {
	testCases := []struct {
		eval     string
		result   bool
		parseErr bool
	}{
		{eval: "if .var33", result: false},
		{eval: "if nil", parseErr: true},
		{eval: "if 0", parseErr: true},
		{eval: "if 0.0", parseErr: true},
		{eval: "if true", result: true},
		{eval: "if false", result: false},
		{eval: `if "hello"`, parseErr: true},

		{eval: "if .var33 == .varFalse", result: false},
		{eval: "if .varFalse != nil", result: true},
		{eval: "if .varNil == 0", result: false},
		{eval: "if .var33 == 0.0", result: false},
		{eval: "if 1.1 == 11", result: false}, // Ideally a constant expression would not be allowed.
		{eval: "if .varFalse == true", result: false},
		{eval: "if .varFalse == false", result: true},
		{eval: `if .varFalse != "hello"`, result: true},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(strings.ReplaceAll(tc.eval, ".", ""), func(t *testing.T) {
			// Parse expression.
			expr, err := New(tc.eval)
			if tc.parseErr {
				if err == nil {
					t.Fatal("expected parse error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if expr == nil {
				t.Fatal("got nil expr")
			}

			// Evaluate
			ctx := &Context{
				vars: map[string]interface{}{
					".var33":    33,
					".varFalse": false,
					".varNil":   nil,
				},
			}
			out, err := expr.Evaluate(ctx)
			if err != nil {
				t.Fatal("failed evaluating expression:", err)
			}

			if out != tc.result {
				t.Fatalf("got = %v, want = %v", out, tc.result)
			}
		})
	}
}

func BenchmarkExpressionEval(b *testing.B) {
	// 2022-11 on Apple M1 Max
	// BenchmarkExpressionEval-10  1648504  709.6 ns/op 344 B/op  16 allocs/op

	// Parse expression.
	expr, err := New(`if .myVar == "192.168.10.11"`)
	if err != nil {
		b.Fatal(err)
	}

	ctx := &Context{
		map[string]interface{}{
			".myVar": "192.168.10.11",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := expr.Evaluate(ctx)
		if err != nil {
			b.Fatal(err)
		}
		if result != true {
			b.Fatal("expected true")
		}
	}
}
