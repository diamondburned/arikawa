package discord

// Poll is a poll attached to a message.
//
// https://discord-api-types.dev/api/next/discord-api-types-v10/interface/APIPoll
// https://discord.com/developers/docs/resources/poll#poll-object
type Poll struct {
	// Question is the question of the poll. Only Text is supported.
	Question PollMedia `json:"question"`
	// Answers are each of the answers available in the poll, up to 10.
	Answers []PollAnswer `json:"answers"`
	// Expiry is the time when the poll ends. It is invalid (null) for
	// non-expiring polls.
	Expiry Timestamp `json:"expiry"`
	// AllowMultiselect specifies whether a user can select multiple answers.
	AllowMultiselect bool `json:"allow_multiselect"`
	// LayoutType is the layout type of the poll.
	LayoutType PollLayoutType `json:"layout_type"`
	// Results contains the results of the poll. It may be nil, in which case
	// the results are unknown and not necessarily zero.
	Results *PollResults `json:"results,omitempty"`
}

// PollCreateRequest is the poll object used when sending a poll in a message.
//
// https://discord-api-types.dev/api/next/discord-api-types-v10/interface/RESTAPIPoll
// https://discord.com/developers/docs/resources/poll#poll-create-request-object
type PollCreateRequest struct {
	// Question is the question of the poll. Only Text is supported.
	Question PollMedia `json:"question"`
	// Answers are each of the answers available in the poll, up to 10. Only
	// PollMedia should be set; AnswerID is assigned by Discord.
	Answers []PollAnswer `json:"answers"`
	// Duration is the number of hours the poll should be open for, up to 32
	// days. Discord defaults to 24 if this is 0.
	Duration int `json:"duration,omitempty"`
	// AllowMultiselect specifies whether a user can select multiple answers.
	AllowMultiselect bool `json:"allow_multiselect,omitempty"`
	// LayoutType is the layout type of the poll. Discord defaults to
	// PollLayoutDefault if this is 0.
	LayoutType PollLayoutType `json:"layout_type,omitempty"`
}

// PollLayoutType is the layout type of a poll.
//
// https://discord-api-types.dev/api/next/discord-api-types-v10/enum/PollLayoutType
// https://discord.com/developers/docs/resources/poll#layout-type
type PollLayoutType int

const (
	// PollLayoutDefault is the default layout type.
	PollLayoutDefault PollLayoutType = 1
)

// PollMedia is the common object used for a poll's question and answers.
//
// https://discord-api-types.dev/api/next/discord-api-types-v10/interface/APIPollMedia
// https://discord.com/developers/docs/resources/poll#poll-media-object
type PollMedia struct {
	// Text is the text of the field. The maximum length is 300 for the
	// question and 55 for any answer.
	Text string `json:"text,omitempty"`
	// Emoji is the emoji of the field. Only ID (for custom emojis) or Name
	// (for Unicode emojis) is set. It is never present for the question.
	Emoji *Emoji `json:"emoji,omitempty"`
}

// PollAnswer is an answer in a poll.
//
// https://discord-api-types.dev/api/next/discord-api-types-v10/interface/APIPollAnswer
// https://discord.com/developers/docs/resources/poll#poll-answer-object
type PollAnswer struct {
	// AnswerID is the ID of the answer. It starts at 1 for the first answer
	// and goes up sequentially. It is only sent by Discord and should be left
	// as 0 when creating a poll.
	AnswerID int `json:"answer_id,omitempty"`
	// PollMedia is the data of the answer.
	PollMedia PollMedia `json:"poll_media"`
}

// PollResults contains the vote counts of a poll.
//
// https://discord-api-types.dev/api/next/discord-api-types-v10/interface/APIPollResults
// https://discord.com/developers/docs/resources/poll#poll-results-object
type PollResults struct {
	// IsFinalized specifies whether the votes have been precisely counted.
	IsFinalized bool `json:"is_finalized"`
	// AnswerCounts are the counts for each answer. An answer with no votes
	// may be omitted.
	AnswerCounts []PollAnswerCount `json:"answer_counts"`
}

// PollAnswerCount is the vote count of a single poll answer.
//
// https://discord-api-types.dev/api/next/discord-api-types-v10/interface/APIPollAnswerCount
// https://discord.com/developers/docs/resources/poll#poll-results-object
type PollAnswerCount struct {
	// ID is the AnswerID of the answer.
	ID int `json:"id"`
	// Count is the number of votes for this answer.
	Count int `json:"count"`
	// MeVoted specifies whether the current user voted for this answer.
	MeVoted bool `json:"me_voted"`
}
