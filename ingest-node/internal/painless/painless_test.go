package painless

import (
	"testing"
)

func TestParseExpression(t *testing.T) {
	vars, err := ParseExpression(`ctx.source?.ip == "192.168.1.1"`)
	if err != nil {
		t.Fatal(err)
	}

	for _, v := range vars {
		t.Log("VAR", v.GetText())
	}
}
