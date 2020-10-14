package complete

import (
	"strings"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/channel/shared"
)

type Completer struct {
	*shared.Channel
}

const MaxCompletion = 15

func New(ch *shared.Channel) cchat.Completer {
	return Completer{ch}
}

// CompleteMessage implements message input completion capability for Discord.
// This method supports user mentions, channel mentions and emojis.
//
// For the individual implementations, refer to channel_completion.go.
func (ch Completer) Complete(words []string, i int64) []cchat.CompletionEntry {
	var word = words[i]
	// Word should have at least a character for the char check.
	if len(word) < 1 {
		return nil
	}

	switch word[0] {
	case '@':
		return ch.CompleteMentions(word[1:])
	case '#':
		return ch.CompleteChannels(word[1:])
	case ':':
		return ch.CompleteEmojis(word[1:])
	}

	return nil
}

func contains(contains string, strs ...string) bool {
	for _, str := range strs {
		if strings.Contains(strings.ToLower(str), contains) {
			return true
		}
	}

	return false
}
