package bot

import "strings"

// Help generates a full Help message. It serves mainly as a reference for
// people to reimplement and change. It doesn't show hidden commands.
func (ctx *Context) Help() string {
	return ctx.HelpGenerate(false)
}

// HelpGenerate generates a full Help message. It serves mainly as a reference
// for people to reimplement and change. If showHidden is true, then hidden
// subcommands and commands will be shown.
func (ctx *Context) HelpGenerate(showHidden bool) string {
	// Generate the header.
	buf := strings.Builder{}
	buf.WriteString("__Help__")

	// Name an
	if ctx.Name != "" {
		buf.WriteString(": " + ctx.Name)
	}
	if ctx.Description != "" {
		buf.WriteString("\n" + IndentLines(ctx.Description))
	}

	// Separators
	buf.WriteString("\n---\n")

	// Generate all commands
	if help := ctx.Subcommand.Help(); help != "" {
		buf.WriteString("__Commands__\n")
		buf.WriteString(IndentLines(help))
		buf.WriteByte('\n')
	}

	var subcommands = ctx.Subcommands()
	var subhelps = make([]string, 0, len(subcommands))

	for _, sub := range subcommands {
		if sub.Hidden && !showHidden {
			continue
		}

		help := sub.HelpShowHidden(showHidden)
		if help == "" {
			continue
		}

		help = IndentLines(help)

		builder := strings.Builder{}
		builder.WriteString("**")
		builder.WriteString(sub.Command)
		builder.WriteString("**")

		for _, alias := range sub.Aliases {
			builder.WriteString("|")
			builder.WriteString("**")
			builder.WriteString(alias)
			builder.WriteString("**")
		}

		if sub.Description != "" {
			builder.WriteString(": ")
			builder.WriteString(sub.Description)
		}

		builder.WriteByte('\n')
		builder.WriteString(help)

		subhelps = append(subhelps, builder.String())
	}

	if len(subhelps) > 0 {
		buf.WriteString("---\n")
		buf.WriteString("__Subcommands__\n")
		buf.WriteString(IndentLines(strings.Join(subhelps, "\n")))
	}

	return buf.String()
}

// IndentLine prefixes every line from input with a single-level indentation.
func IndentLines(input string) string {
	const indent = "      "
	var lines = strings.Split(input, "\n")
	for i := range lines {
		lines[i] = indent + lines[i]
	}
	return strings.Join(lines, "\n")
}
