package main

import "testing"

func expectFlattenWhere(t *testing.T, input []string, expected string) {
	wb := whereBuilder{
		pieces: input,
		args:   nil,
	}
	result := wb.flatten()
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestFlattenWhere(t *testing.T) {
	expectFlattenWhere(t, []string{}, "")
	expectFlattenWhere(t, []string{"A"}, "A")
	expectFlattenWhere(t, []string{"A", "B"}, "(A OR B)")
	expectFlattenWhere(t, []string{"A", "", "B"}, "A AND B")
	expectFlattenWhere(t, []string{"A", "B", "", "C"}, "(A OR B) AND C")
	expectFlattenWhere(t, []string{"A", "", "B", "C"}, "A AND (B OR C)")
}

func expectLikePattern(t *testing.T, input string, expected string) {
	result := likePattern(input)
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestLikePattern(t *testing.T) {
	expectLikePattern(t, ``, ``)
	expectLikePattern(t, `_`, `\_`)
	expectLikePattern(t, `%`, `\%`)
	expectLikePattern(t, `*`, `%`)
	expectLikePattern(t, `?`, `_`)
	expectLikePattern(t, `Foo-*-Bar_%`, `Foo-%-Bar\_\%`)
}

func expectWhereClause(t *testing.T, query *Query, expected string, expectedArgs ...interface{}) {
	result, args := makeWhereClause(query)
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
	argsMatch := len(expectedArgs) == len(args)
	if argsMatch {
		for i := range args {
			if args[i] != expectedArgs[i] {
				argsMatch = false
			}
		}
	}
	if !argsMatch {
		t.Errorf("Expected '%+v', got '%+v'", expectedArgs, args)
	}
}

func TestMakeWhereClause(t *testing.T) {
	expectWhereClause(t, NewQuery().Repository("foo/bar"),
		" WHERE t.repository = $1",
		"foo/bar")
	expectWhereClause(t, NewQuery().Tag("foo"),
		" WHERE t.tag = $1",
		"foo")
	expectWhereClause(t, NewQuery().OS("foo"),
		" WHERE i.os = $1",
		"foo")
	expectWhereClause(t, NewQuery().Arch("foo"),
		" WHERE i.arch = $1",
		"foo")
	expectWhereClause(t, NewQuery().AnnotationIs("org.fishsoup.nonsense", "foo"),
		" WHERE i.annotations ? $1",
		"org.fishsoup.nonsense")
	expectWhereClause(t, NewQuery().AnnotationExists("org.fishsoup.nonsense"),
		" WHERE i.annotations @> $1",
		`{"org.fishsoup.nonsense":""}`)
	expectWhereClause(t, NewQuery().AnnotationMatches("org.fishsoup.nonsense", "foo-*"),
		" WHERE jsonb_object_field_text(i.annotations, $1) like $2",
		"org.fishsoup.nonsense", "foo-%")

	expectWhereClause(t, NewQuery(), "")
	expectWhereClause(t, NewQuery().Repository("foo").Repository("bar"),
		" WHERE (t.repository = $1 OR t.repository = $2)",
		"foo", "bar")
	expectWhereClause(t, NewQuery().Repository("foo").Tag("bar").Tag("baz"),
		" WHERE t.repository = $1 AND (t.tag = $2 OR t.tag = $3)",
		"foo", "bar", "baz")
}
