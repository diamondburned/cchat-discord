package completer

import (
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/state"
	"github.com/diamondburned/cchat-discord/internal/urlutils"
	"github.com/diamondburned/cchat/text"
)

func (ch ChannelCompleter) CompleteEmojis(word string) (entries []cchat.CompletionEntry) {
	return Emojis(ch.State, ch.GuildID, word)
}

func Emojis(s *state.Instance, gID discord.GuildID, word string) []cchat.CompletionEntry {
	// Ignore if empty word.
	if word == "" {
		return nil
	}

	guilds, err := s.EmojiState.Get(gID)
	if err != nil {
		return nil
	}

	var entries []cchat.CompletionEntry
	var distances map[string]int

GuildSearch:
	for _, guild := range guilds {
		for _, emoji := range guild.Emojis {
			rank := rankFunc(word, emoji.Name)
			if rank == -1 {
				continue
			}

			// Defer allocation until we've found something.
			ensureEntriesMade(&entries)
			ensureDistancesMade(&distances)

			raw := emoji.String()

			entries = append(entries, cchat.CompletionEntry{
				Raw:       raw,
				Text:      text.Rich{Content: ":" + emoji.Name + ":"},
				Secondary: text.Rich{Content: guild.Name},
				IconURL:   urlutils.Sized(emoji.EmojiURL(), 64),
				Image:     true,
			})

			distances[raw] = rank

			if len(entries) >= MaxCompletion {
				break GuildSearch
			}
		}
	}

	sortDistances(entries, distances)
	return entries
}
