package complete

import (
	"strings"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat/text"
)

func (ch Completer) CompleteChannels(word string) (entries []cchat.CompletionEntry) {
	// Ignore if empty word.
	if word == "" {
		return
	}

	// Ignore if we're not in a guild.
	if !ch.GuildID.IsValid() {
		return
	}

	c, err := ch.State.Store.Channels(ch.GuildID)
	if err != nil {
		return
	}

	var match = strings.ToLower(word)

	for _, channel := range c {
		if !contains(match, channel.Name) {
			continue
		}

		var category string
		if channel.CategoryID.IsValid() {
			if c, _ := ch.State.Store.Channel(channel.CategoryID); c != nil {
				category = c.Name
			}
		}

		entries = append(entries, cchat.CompletionEntry{
			Raw:       channel.Mention(),
			Text:      text.Rich{Content: "#" + channel.Name},
			Secondary: text.Rich{Content: category},
		})

		if len(entries) >= MaxCompletion {
			return
		}
	}

	return
}
