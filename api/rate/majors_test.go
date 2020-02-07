// +build unit

package rate

import "testing"

func TestBucketKey(t *testing.T) {
	var tests = [][2]string{
		{"/guilds/123123/messages",
			"/guilds/123123/messages"},
		{"/guilds/123123/",
			"/guilds/123123/"},
		{"/channels/123131231",
			"/channels/123131231"},
		{"/channels/123123/message/123456",
			"/channels/123123/message/"},
		{"/user/123123", "/user/"},
		// Not sure about this:
		{"/user/123123/", "/user//"},
		{"/channels/1/messages/1/reactions/🤔/@me",
			"/channels/1/messages//reactions//@me"},
		{"/channels/1/messages/2/reactions/thonk:123123/@me",
			"/channels/1/messages//reactions//@me"},
		// Actual URL:
		{"/channels/486833611564253186/messages/540519319814275089/reactions/🥺/@me",
			"/channels/486833611564253186/messages//reactions//@me"},
	}

	for _, conds := range tests {
		key := ParseBucketKey(conds[0])
		if key != conds[1] {
			t.Fatalf("Expected/got\n%s\n%s", conds[1], key)
		}
	}
}
