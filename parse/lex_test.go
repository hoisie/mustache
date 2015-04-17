package parse

import "testing"

type lexTest struct {
	name  string
	input string
	items []item
}

var (
	tEOF          = item{itemEOF, 0, ""}
	tLeft         = item{itemLeftDelim, 0, "{{"}
	tRight        = item{itemRightDelim, 0, "}}"}
	tLeftSection  = item{itemLeftSectionDelim, 0, "{{#"}
	tRightSection = item{itemRightSectionDelim, 0, "{{/"}
)

var lexTests = []lexTest{
	{"empty", "", []item{tEOF}},
	{"numbers", "12345", []item{{itemText, 0, "12345"}, tEOF}},
	{"spaces", " \t\n", []item{{itemText, 0, " \t\n"}, tEOF}},
	{"text", `now is the time`, []item{{itemText, 0, "now is the time"}, tEOF}},
	{"text with comment", "12345{{! Comment Block! }}67890", []item{
		{itemText, 0, "12345"},
		{itemComment, 0, " Comment Block! "},
		{itemText, 0, "67890"},
		tEOF,
	}},
	{"text with multi-line comment", "12345{{!\n  This is a\n  multi-line comment...\n}}67890\n", []item{
		{itemText, 0, "12345"},
		{itemComment, 0, "\n  This is a\n  multi-line comment...\n"},
		{itemText, 0, "67890\n"},
		tEOF,
	}},
	{"text with standalone comment", "Begin.\n{{! Comment Block! }}\nEnd.\n", []item{
		{itemText, 0, "Begin.\n"},
		{itemComment, 0, " Comment Block! "},
		{itemText, 0, "\nEnd.\n"},
		tEOF,
	}},
	{"text with indented standalone comment", "Begin.\n  {{! Indented Comment Block! }}\nEnd.\n", []item{
		{itemText, 0, "Begin.\n  "},
		{itemComment, 0, " Indented Comment Block! "},
		{itemText, 0, "\nEnd.\n"},
		tEOF,
	}},
	{"interpolation", "{{foo}}", []item{
		tLeft,
		{itemVariable, 0, "foo"},
		tRight,
		tEOF,
	}},
	{"section", "{{#foo}}stuff goes here{{/foo}}", []item{
		tLeftSection,
		{itemVariable, 0, "foo"},
		tRight,
		{itemText, 0, "stuff goes here"},
		tRightSection,
		{itemVariable, 0, "foo"},
		tRight,
		tEOF,
	}},
	{"partial", "{{>text}}", []item{
		tLeft,
		{itemPartial, 0, "text"},
		tRight,
		tEOF,
	}},
}

func TestLex(t *testing.T) {
	for _, test := range lexTests {
		items := collect(&test, "", "")
		if !equal(items, test.items, false) {
			t.Errorf("%s: got\n\t%+v\nexpected\n\t%v", test.name, items, test.items)
		}
	}
}

// collect gathers the emitted items into a slice.
func collect(t *lexTest, left, right string) (items []item) {
	l := lex(t.name, t.input, left, right)
	for {
		item := l.nextItem()
		items = append(items, item)
		if item.typ == itemEOF || item.typ == itemError {
			break
		}
	}
	return
}

func equal(i1, i2 []item, checkPos bool) bool {
	if len(i1) != len(i2) {
		return false
	}
	for k := range i1 {
		if i1[k].typ != i2[k].typ {
			return false
		}
		if i1[k].val != i2[k].val {
			return false
		}
		if checkPos && i1[k].pos != i2[k].pos {
			return false
		}
	}
	return true
}
