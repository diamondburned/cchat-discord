package completer

import (
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/state"
	"github.com/diamondburned/cchat-discord/internal/segments/mention"
	"github.com/diamondburned/cchat/text"
)

func (ch ChannelCompleter) CompleteChannels(word string) []cchat.CompletionEntry {
	// Ignore if empty word.
	if word == "" {
		return nil
	}

	// Ignore if we're not in a guild.
	if !ch.GuildID.IsValid() {
		return nil
	}

	c, err := ch.State.Cabinet.Channels(ch.GuildID)
	if err != nil {
		return nil
	}

	return completeChannels(c, word, ch.State)
}

func DMChannels(s *state.Instance, word string) []cchat.CompletionEntry {
	channels, err := s.Cabinet.PrivateChannels()
	if err != nil {
		return nil
	}
	// We only need the state to look for categories, which is never the case
	// for private channels.
	return completeChannels(channels, word, nil)
}

func rankChannel(word string, ch discord.Channel) int {
	switch ch.Type {
	case discord.GroupDM, discord.DirectMessage:
		return rankFunc(word, ch.Name+" "+mention.ChannelName(ch))
	default:
		return rankFunc(word, ch.Name)
	}
}

func completeChannels(
	channels []discord.Channel, word string, s *state.Instance) []cchat.CompletionEntry {

	var entries []cchat.CompletionEntry
	var distances map[string]int

	for _, channel := range channels {
		rank := rankChannel(word, channel)
		if rank == -1 {
			continue
		}

		var category string
		if s != nil && channel.CategoryID.IsValid() {
			if cat, _ := s.Cabinet.Channel(channel.CategoryID); cat != nil {
				category = cat.Name
			}
		}

		// Defer allocation until we've found something.
		ensureEntriesMade(&entries)
		ensureDistancesMade(&distances)

		raw := channel.Mention()

		entries = append(entries, cchat.CompletionEntry{
			Raw:       raw,
			Text:      text.Plain("#" + channel.Name),
			Secondary: text.Plain(category),
		})

		distances[raw] = rank

		if len(entries) >= MaxCompletion {
			break
		}
	}

	sortDistances(entries, distances)
	return entries
}
