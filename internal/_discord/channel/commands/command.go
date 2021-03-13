package commands

import (
	"bytes"

	"github.com/diamondburned/cchat-discord/internal/discord/channel/shared"
)

type Command struct {
	Name    string
	Args    Arguments
	Desc    string
	RunFunc func(shared.Channel, []string) ([]byte, error) // words[1:]
}

func (cmd Command) writeHelp(builder *bytes.Buffer) {
	builder.WriteString(cmd.Name)
	cmd.Args.writeHelp(builder)

	if cmd.Desc != "" {
		builder.WriteString("\n\t")
		builder.WriteString(cmd.Desc)
	}
}
