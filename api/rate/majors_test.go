package rate

import "testing"

func TestBucketKey(t *testing.T) {
	var tests = [][2]string{
		{"/guilds/123123/messages", "/guilds/123123/messages"},
		{"/guilds/123123/", "/guilds/123123/"},
		{"/channels/123131231", "/channels/123131231"},
		{"/channels/123123/message/123456", "/channels/123123/message/"},
		{"/user/123123", "/user/"},
		{"/user/123123/", "/user//"}, // not sure about this
	}

	for _, conds := range tests {
		key := ParseBucketKey(conds[0])
		if key != conds[1] {
			t.Fatalf("Expected/got\n%s\n%s", conds[1], key)
		}
	}
}
