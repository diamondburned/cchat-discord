package complete

import (
	"strings"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/state"
	"github.com/diamondburned/cchat-discord/internal/urlutils"
	"github.com/diamondburned/cchat/text"
)

func (ch Completer) CompleteEmojis(word string) (entries []cchat.CompletionEntry) {
	return CompleteEmojis(ch.State, ch.GuildID, word)
}

func CompleteEmojis(s *state.Instance, gID discord.GuildID, word string) []cchat.CompletionEntry {
	// Ignore if empty word.
	if word == "" {
		return nil
	}

	e, err := s.EmojiState.Get(gID)
	if err != nil {
		return nil
	}

	var match = strings.ToLower(word)
	var entries = make([]cchat.CompletionEntry, 0, MaxCompletion)

	for _, guild := range e {
		for _, emoji := range guild.Emojis {
			if !contains(match, emoji.Name) {
				continue
			}

			entries = append(entries, cchat.CompletionEntry{
				Raw:       emoji.String(),
				Text:      text.Rich{Content: ":" + emoji.Name + ":"},
				Secondary: text.Rich{Content: guild.Name},
				IconURL:   urlutils.Sized(emoji.EmojiURL(), 32), // small
				Image:     true,
			})

			if len(entries) >= MaxCompletion {
				return entries
			}
		}
	}

	return entries
}
