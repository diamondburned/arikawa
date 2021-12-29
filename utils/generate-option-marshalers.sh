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
	NumberOption
)

for ((i = 0; i < ${#types[@]}; i++)); {
	recv=$(head -c1 <<< "${types[$i]}" | tr "[:upper:]" "[:lower:]")

	cat<<EOF
// MarshalJSON marshals ${types[$i]} to JSON with the "type" field.
func (${recv} *${types[$i]}) MarshalJSON() ([]byte, error) {
	type raw ${types[$i]}
	return json.Marshal(struct {
		Type CommandOptionType \`json:"type"\`
		*raw
	}{
		Type: ${recv}.Type(),
		raw:  (*raw)(${recv}),
	})
}

EOF
}
