package api

import (
	"strconv"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/httputil"
)

// MaxPollAnswerVotersFetchLimit is the maximum number of users that can be
// fetched with a single PollAnswerVoters call.
const MaxPollAnswerVotersFetchLimit = 100

// PollAnswerVotersData is the query data for PollAnswerVoters.
//
// https://discord-api-types.dev/api/next/discord-api-types-v10/interface/RESTGetAPIPollAnswerVotersQuery
type PollAnswerVotersData struct {
	// After gets users after this user ID.
	After discord.UserID `schema:"after,omitempty"`
	// Limit is the max number of users to return (1-100). Discord defaults to
	// 25 if this is 0.
	Limit uint `schema:"limit,omitempty"`
}

// PollAnswerVoters returns a list of users that voted for the given answer in
// the poll of the given message. This makes a single request; use data.After
// to paginate.
//
// https://discord-api-types.dev/api/next/discord-api-types-v10/interface/RESTGetAPIPollAnswerVotersResult
// https://discord.com/developers/docs/resources/poll#get-answer-voters
func (c *Client) PollAnswerVoters(
	channelID discord.ChannelID, messageID discord.MessageID,
	answerID int, data PollAnswerVotersData) ([]discord.User, error) {

	if data.Limit > MaxPollAnswerVotersFetchLimit {
		data.Limit = MaxPollAnswerVotersFetchLimit
	}

	var resp struct {
		Users []discord.User `json:"users"`
	}

	err := c.RequestJSON(
		&resp, "GET",
		EndpointChannels+channelID.String()+
			"/polls/"+messageID.String()+
			"/answers/"+strconv.Itoa(answerID),
		httputil.WithSchema(c, data),
	)
	return resp.Users, err
}

// ExpirePoll immediately ends the poll of the given message. Polls created by
// another user cannot be expired. It returns the updated message and fires a
// Message Update gateway event.
//
// https://discord.com/developers/docs/resources/poll#end-poll
func (c *Client) ExpirePoll(
	channelID discord.ChannelID, messageID discord.MessageID) (*discord.Message, error) {

	var msg *discord.Message
	return msg, c.RequestJSON(
		&msg, "POST",
		EndpointChannels+channelID.String()+
			"/polls/"+messageID.String()+"/expire",
	)
}

// VotePoll sets the current user's votes in the poll of the given message to
// the given answer IDs, replacing any previous votes. Passing no answer IDs
// removes the user's votes. This endpoint is only available to user accounts.
//
// https://docs.discord.food/resources/message#create-poll-vote
func (c *Client) VotePoll(
	channelID discord.ChannelID, messageID discord.MessageID, answerIDs ...int) error {

	if answerIDs == nil {
		answerIDs = []int{}
	}

	body := struct {
		AnswerIDs []int `json:"answer_ids"`
	}{answerIDs}

	return c.FastRequest(
		"PUT",
		EndpointChannels+channelID.String()+
			"/polls/"+messageID.String()+"/answers/@me",
		httputil.WithJSONBody(body),
	)
}
