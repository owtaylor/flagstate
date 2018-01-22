package web

import (
	"testing"
)

func stringsEqual(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func expectParseIfMatch(t *testing.T, input string, expected []string) {
	res, err := ParseIfMatch(input)
	if err != nil {
		t.Error(err)
		return
	}
	if !stringsEqual(res, expected) {
		t.Errorf("parsing %v, expected %+v, got %+v", input, expected, res)
	}
}

func expectParseIfMatchError(t *testing.T, input string) {
	res, err := ParseIfMatch(input)
	if err == nil {
		t.Errorf("Parsing %v, expected error, got %v", t, res)
	}
}

func TestParseIfMatch(t *testing.T) {
	expectParseIfMatch(t,
		`"foo",*,W/"bar"`,
		[]string{`"foo"`, `*`, `W/"bar"`},
	)
	expectParseIfMatch(t,
		` "foo" , * , W/"bar" `,
		[]string{`"foo"`, `*`, `W/"bar"`},
	)
	expectParseIfMatch(t,
		` "foo\"bar", "baz" `,
		[]string{`"foo\"bar"`, `"baz"`},
	)
	expectParseIfMatch(t,
		`,"foo",,"bar",`,
		[]string{`"foo"`, `"bar"`},
	)
	expectParseIfMatch(t,
		`""`,
		[]string{`""`},
	)

	expectParseIfMatchError(t, `foo`)
	expectParseIfMatchError(t, `"foo`)
	expectParseIfMatchError(t, `"foo" "bar"`)
}
