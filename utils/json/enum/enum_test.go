package enum

import (
	"reflect"
	"testing"
)

func TestInt8ToJSON(t *testing.T) {
	testCases := []struct {
		name   string
		src    Enum
		expect []byte
	}{
		{
			name:   "null",
			src:    Null,
			expect: []byte("null"),
		},
		{
			name:   "value",
			src:    12,
			expect: []byte("12"),
		},
	}

	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			actual := ToJSON(c.src)

			if !reflect.DeepEqual(actual, c.expect) {
				t.Errorf("expected nullable.Int8ToJSON to return: %+v, but got: %+v", c.expect, actual)
			}
		})
	}
}

func TestInt8FromJSON(t *testing.T) {
	testCases := []struct {
		name   string
		src    []byte
		expect Enum
		err    bool
	}{
		{
			name:   "null",
			src:    []byte("null"),
			expect: Null,
			err:    false,
		},
		{
			name:   "value",
			src:    []byte("12"),
			expect: 12,
			err:    false,
		},
		{
			name:   "invalid input",
			src:    []byte("NaN"),
			expect: 0,
			err:    true,
		},
	}

	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			actual, err := FromJSON(c.src)

			if c.err {
				if err == nil {
					t.Error("expected nullable.Int8FromJSON to return an error, but it did not")
				}
			} else {
				if !reflect.DeepEqual(actual, c.expect) {
					t.Errorf("expected nullable.Int8FromJSON to return: %+v, but got: %+v", c.expect, actual)
				}

				if err != nil {
					t.Errorf("nullable.Int8FromJSON returned an error: %s", err.Error())
				}
			}
		})
	}
}
