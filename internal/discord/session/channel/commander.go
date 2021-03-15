package channel

import (
	"strings"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/session/channel/commands"
	"github.com/diamondburned/cchat-discord/internal/discord/session/channel/messenger/sender/completer"
	"github.com/diamondburned/cchat-discord/internal/discord/session/channel/shared"
	"github.com/diamondburned/cchat/text"
)

type Commander struct {
	shared.Channel
	msgCompl completer.ChannelCompleter
}

func NewCommander(ch shared.Channel) cchat.Commander {
	return Commander{
		Channel: ch,
		msgCompl: completer.ChannelCompleter{
			Channel: ch,
		},
	}
}

func (ch Commander) AsCompleter() cchat.Completer { return ch }

func (ch Commander) Run(words []string) ([]byte, error) {
	return commands.World.Run(ch.Channel, words)
}

func (ch Commander) Complete(words []string, i int64) []cchat.CompletionEntry {
	if i == 0 {
		commands := commands.World.Find(words[0])

		var entries = make([]cchat.CompletionEntry, 0, len(commands))
		if strings.HasPrefix(words[0], "help") {
			entries = append(entries, cchat.CompletionEntry{
				Raw:       "help",
				Text:      text.Plain("help"),
				Secondary: text.Plain("Prints the help message"),
			})
		}

		for _, cmd := range commands {
			entries = append(entries, cchat.CompletionEntry{
				Raw:       cmd.Name,
				Text:      text.Plain(cmd.Name),
				Secondary: text.Plain(cmd.Desc),
			})
		}

		return entries
	}

	cmd := commands.World.FindExact(words[0])
	if cmd == nil {
		return nil
	}

	name, _ := cmd.Args.At(int(i) - 1)
	if name == "" {
		return nil
	}

	switch name {
	case "mention:user":
		return ch.msgCompl.CompleteMentions(words[i])
	case "mention:emoji":
		return ch.msgCompl.CompleteEmojis(words[i])
	case "mention:channel":
		return ch.msgCompl.CompleteChannels(words[i])
	}

	return nil
}
