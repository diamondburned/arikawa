package bot

import (
	"reflect"
	"strings"
	"testing"
)

type mockParser string

func (m *mockParser) Parse(s string) error {
	*m = mockParser(s)
	return nil
}

func mockParse(str string) *mockParser {
	return (*mockParser)(&str)
}

func TestArguments(t *testing.T) {
	testArgs(t, "string", "string")
	testArgs(t, true, "true")
	testArgs(t, false, "n")
	testArgs(t, int64(69420), "69420")
	testArgs(t, uint64(1337), "1337")
	testArgs(t, 69.420, "69.420")
	testArgs(t, mockParse("testString"), "testString")
	testArgs(t, *mockParse("testString"), "testString")

	_, err := newArgument(reflect.TypeOf(struct{}{}), false)
	if !strings.HasPrefix(err.Error(), "invalid type: ") {
		t.Fatal("Unexpected error:", err)
	}
}

func testArgs(t *testing.T, expect interface{}, input string) {
	f, err := newArgument(reflect.TypeOf(expect), false)
	if err != nil {
		t.Fatal("Failed to get argument value function:", err)
	}

	v, err := f.fn(input)
	if err != nil {
		t.Fatal("avfs returned with error:", err)
	}

	if v := v.Interface(); !reflect.DeepEqual(v, expect) {
		t.Fatal("Value  :", v, "\nExpects:", expect)
	}
}

// used for ctx_test.go

type customParsed struct {
	parsed bool
}

func (c *customParsed) Parse(string) error {
	c.parsed = true
	return nil
}
