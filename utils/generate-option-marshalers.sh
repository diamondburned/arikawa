#!/usr/bin/env bash

types=(
	SubcommandOption
	SubcommandGroupOption
	StringOption
	IntegerOption
	BooleanOption
	UserOption
	ChannelOption
	RoleOption
	MentionableOption
)

recvs=(
	s
	s
	s
	i
	b
	u
	c
	r
	m
)

for ((i = 0; i < 6; i++)); {
	cat<<EOF
// MarshalJSON marshals ${types[$i]} to JSON with the "type" field.
func (${recvs[$i]} *${types[$i]}) MarshalJSON() ([]byte, error) {
	type raw ${types[$i]}
	return json.Marshal(struct {
		Type CommandOptionType \`json:"type"\`
		*raw
	}{
		Type: ${recvs[$i]}.Type(),
		raw:  (*raw)(${recvs[$i]}),
	})
}

EOF
}
