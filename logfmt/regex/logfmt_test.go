package regex

import (
	"reflect"
	"strings"
	"testing"

	"github.com/andrewkroh/go-examples/logfmt/internal/validate"
)

func TestParse(t *testing.T) {
	for _, tc := range validate.MessageTestCases {
		tc := tc
		t.Run(tc.Msg, func(t *testing.T) {
			msg, err := Parse(tc.Msg)
			if err != nil {
				t.Fatal(err)
			}

			observed := make([]string, 0, len(msg.KeyValuePairs)*2)
			for _, p := range msg.KeyValuePairs {
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

func BenchmarkParse(b *testing.B) {
	// BenchmarkParse-10  433690  2758 ns/op  1391 B/op  16 allocs/op

	const msg = `level=info msg="Stopping all fetchers" tag=stopping_fetchers id=ConsumerFetcherManager-1382721708341 module=kafka.consumer.ConsumerFetcherManager`

	for i := 0; i < b.N; i++ {
		_, err := Parse(msg)
		if err != nil {
			b.Fatal(err)
		}
	}
}
