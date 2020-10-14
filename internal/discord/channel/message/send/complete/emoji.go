package complete

import (
	"strings"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/urlutils"
	"github.com/diamondburned/cchat/text"
)

func (ch Completer) CompleteEmojis(word string) (entries []cchat.CompletionEntry) {
	// Ignore if empty word.
	if word == "" {
		return
	}

	e, err := ch.State.EmojiState.Get(ch.GuildID)
	if err != nil {
		return
	}

	var match = strings.ToLower(word)

	for _, guild := range e {
		for _, emoji := range guild.Emojis {
			if contains(match, emoji.Name) {
				entries = append(entries, cchat.CompletionEntry{
					Raw:       emoji.String(),
					Text:      text.Rich{Content: ":" + emoji.Name + ":"},
					Secondary: text.Rich{Content: guild.Name},
					IconURL:   urlutils.Sized(emoji.EmojiURL(), 32), // small
					Image:     true,
				})
				if len(entries) >= MaxCompletion {
					return
				}
			}
		}
	}

	return
}
