package shellwords

import (
	"reflect"
	"testing"
)

type wordsTest struct {
	line  string
	args  []string
	doErr bool
}

func TestParse(t *testing.T) {
	var tests = []wordsTest{
		{
			"",
			nil,
			false,
		},
		{
			"'",
			nil,
			true,
		},
		{
			`this is a "te""st"`,
			[]string{"this", "is", "a", "test"},
			false,
		},
		{
			`hanging "quote`,
			[]string{"hanging", "quote"},
			true,
		},
		{
			`Hello,　世界`,
			[]string{"Hello,", "世界"},
			false,
		},
		{
			"this is `inline code`",
			[]string{"this", "is", "inline code"},
			false,
		},
		{
			"how about a ```go\npackage main\n```\ngo code?",
			[]string{"how", "about", "a", "go\npackage main\n", "go", "code?"},
			false,
		},
		{
			"this should not crash `",
			[]string{"this", "should", "not", "crash"},
			true,
		},
		{
			"this should not crash '",
			[]string{"this", "should", "not", "crash"},
			true,
		},
		{
			"iPhone “double quoted” text",
			[]string{"iPhone", "double quoted", "text"},
			true,
		},
		{
			"iPhone ‘single quoted’ text",
			[]string{"iPhone", "single quoted", "text"},
			true,
		},
	}

	for _, test := range tests {
		w, err := Parse(test.line)
		if err != nil && !test.doErr {
			t.Errorf("Error at %q: %v", test.line, err)
			continue
		}

		if !reflect.DeepEqual(w, test.args) {
			t.Errorf("Inequality:\n%#v !=\n%#v", w, test.args)
		}
	}
}
