package discord

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestPollJSON(t *testing.T) {
	tests := []struct {
		name string
		new  func() any
		json string
	}{
		{
			name: "poll",
			new:  func() any { return new(Poll) },
			json: `{
				"question": {"text": "Which one?"},
				"answers": [
					{"answer_id": 1, "poll_media": {"text": "Cats", "emoji": {"id": null, "name": "🐱"}}},
					{"answer_id": 2, "poll_media": {"text": "Dogs", "emoji": {"id": "41771983429993937", "name": "dog", "animated": true}}}
				],
				"expiry": "2024-04-01T12:00:00Z",
				"allow_multiselect": true,
				"layout_type": 1,
				"results": {
					"is_finalized": false,
					"answer_counts": [{"id": 1, "count": 3, "me_voted": true}]
				}
			}`,
		},
		{
			name: "poll without expiry or results",
			new:  func() any { return new(Poll) },
			json: `{
				"question": {"text": "Forever?"},
				"answers": [{"answer_id": 1, "poll_media": {"text": "Yes"}}],
				"expiry": null,
				"allow_multiselect": false,
				"layout_type": 1
			}`,
		},
		{
			name: "create request",
			new:  func() any { return new(PollCreateRequest) },
			json: `{
				"question": {"text": "Lunch?"},
				"answers": [
					{"poll_media": {"text": "Pizza"}},
					{"poll_media": {"text": "Sushi", "emoji": {"id": null, "name": "🍣"}}}
				],
				"duration": 48,
				"layout_type": 1
			}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			v := test.new()
			if err := json.Unmarshal([]byte(test.json), v); err != nil {
				t.Fatal("failed to unmarshal:", err)
			}

			b, err := json.Marshal(v)
			if err != nil {
				t.Fatal("failed to marshal:", err)
			}

			var got, want any
			if err := json.Unmarshal(b, &got); err != nil {
				t.Fatal("failed to unmarshal marshaled JSON:", err)
			}
			if err := json.Unmarshal([]byte(test.json), &want); err != nil {
				t.Fatal("failed to unmarshal test JSON:", err)
			}

			if !reflect.DeepEqual(got, want) {
				t.Errorf("round trip mismatch:\ngot  %s\nwant %s", b, test.json)
			}
		})
	}
}
