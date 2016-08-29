package matcher

import (
	"bytes"
	"testing"

	"github.com/turbinelabs/test/assert"
)

func TestAnyWriter(t *testing.T) {
	aw := AnyWriter{[]byte("yep")}
	buf := &bytes.Buffer{}

	assert.False(t, aw.Matches("nope"))

	assert.True(t, aw.Matches(buf))
	assert.Equal(t, buf.String(), "yep")

	assert.Equal(t, aw.String(), `AnyWriter("yep")`)
}

func TestPredicateMatcher(t *testing.T) {
	eqm := PredicateMatcher{
		Name: "string value tester",
		Test: func(x interface{}) bool {
			v, ok := x.(string)
			if !ok {
				return false
			}

			return v == "matched!"
		},
	}

	assert.False(t, eqm.Matches("nope"))

	assert.True(t, eqm.Matches("matched!"))
	assert.Equal(t, eqm.String(), "PredicateMatcher(string value tester)")
}
