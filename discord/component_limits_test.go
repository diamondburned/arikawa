package discord

import (
	"testing"

	"github.com/diamondburned/arikawa/v3/utils/json"
)

func TestComponentLimitsUnmarshal(t *testing.T) {
	var sel StringSelectComponent
	if err := json.Unmarshal([]byte(`{"custom_id":"a","options":[],"max_values":3}`), &sel); err != nil {
		t.Fatal(err)
	}
	if sel.ValueLimits != [2]int{1, 3} {
		t.Errorf("select limits = %v, want [1 3]", sel.ValueLimits)
	}

	var input TextInputComponent
	if err := json.Unmarshal([]byte(`{"custom_id":"a","style":1,"min_length":2,"max_length":32}`), &input); err != nil {
		t.Fatal(err)
	}
	if input.LengthLimits != [2]int{2, 32} || input.CustomID != "a" {
		t.Errorf("text input = %+v", input)
	}

	var button ButtonComponent
	if err := json.Unmarshal([]byte(`{"style":5,"label":"x","url":"https://example.com"}`), &button); err != nil {
		t.Fatal(err)
	}
	if button.LinkURL() != "https://example.com" {
		t.Errorf("link URL = %q", button.LinkURL())
	}
}
