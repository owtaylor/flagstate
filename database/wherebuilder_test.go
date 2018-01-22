package database

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
		" WHERE t.Repository = $1",
		"foo/bar")
	expectWhereClause(t, NewQuery().Tag("foo"),
		" WHERE t.Tag = $1",
		"foo")
	expectWhereClause(t, NewQuery().OS("foo"),
		" WHERE i.OS = $1",
		"foo")
	expectWhereClause(t, NewQuery().Architecture("foo"),
		" WHERE i.Architecture = $1",
		"foo")
	expectWhereClause(t, NewQuery().AnnotationIs("org.fishsoup.nonsense", "foo"),
		" WHERE i.Annotations @> $1",
		`{"org.fishsoup.nonsense":"foo"}`)
	expectWhereClause(t, NewQuery().AnnotationExists("org.fishsoup.nonsense"),
		" WHERE i.Annotations ? $1",
		"org.fishsoup.nonsense")
	expectWhereClause(t, NewQuery().AnnotationMatches("org.fishsoup.nonsense", "foo-*"),
		" WHERE jsonb_object_field_text(i.Annotations, $1) like $2",
		"org.fishsoup.nonsense", "foo-%")
	expectWhereClause(t, NewQuery().LabelIs("org.fishsoup.nonsense", "foo"),
		" WHERE i.Labels @> $1",
		`{"org.fishsoup.nonsense":"foo"}`)
	expectWhereClause(t, NewQuery().LabelExists("org.fishsoup.nonsense"),
		" WHERE i.Labels ? $1",
		"org.fishsoup.nonsense")
	expectWhereClause(t, NewQuery().LabelMatches("org.fishsoup.nonsense", "foo-*"),
		" WHERE jsonb_object_field_text(i.Labels, $1) like $2",
		"org.fishsoup.nonsense", "foo-%")

	expectWhereClause(t, NewQuery(), "")
	expectWhereClause(t, NewQuery().Repository("foo").Repository("bar"),
		" WHERE (t.Repository = $1 OR t.Repository = $2)",
		"foo", "bar")
	expectWhereClause(t, NewQuery().Repository("foo").Tag("bar").Tag("baz"),
		" WHERE t.Repository = $1 AND (t.Tag = $2 OR t.Tag = $3)",
		"foo", "bar", "baz")
}
