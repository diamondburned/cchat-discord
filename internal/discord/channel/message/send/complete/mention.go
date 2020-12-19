package complete

import (
	"strings"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/message"
	"github.com/diamondburned/cchat-discord/internal/discord/state"
	"github.com/diamondburned/cchat/text"
)

// MessageMentions generates a list of user mention completion entries from
// messages.
func MessageMentions(msgs []discord.Message) []cchat.CompletionEntry {
	return GuildMessageMentions(msgs, nil, nil)
}

// GuildMessageMentions generates a list of member mention completion entries
// from guild messages.
func GuildMessageMentions(
	msgs []discord.Message,
	state *state.Instance, guild *discord.Guild) []cchat.CompletionEntry {

	if len(msgs) == 0 {
		return nil
	}

	// Keep track of the number of authors.
	// TODO: fix excess allocations

	var entries []cchat.CompletionEntry
	var authors map[discord.UserID]struct{}

	for _, msg := range msgs {
		// If we've already added the author into the list, then skip.
		if _, ok := authors[msg.Author.ID]; ok {
			continue
		}

		ensureAuthorMapMade(&authors)
		authors[msg.Author.ID] = struct{}{}

		var rich text.Rich

		if guild != nil && state != nil {
			m, err := state.Store.Member(guild.ID, msg.Author.ID)
			if err == nil {
				rich = message.RenderMemberName(*m, *guild, state)
			}
		}

		// Fallback to searching the author if member fails.
		if rich.IsEmpty() {
			rich = text.Plain(msg.Author.Username)
		}

		ensureEntriesMade(&entries)

		entries = append(entries, cchat.CompletionEntry{
			Raw:       msg.Author.Mention(),
			Text:      rich,
			Secondary: text.Plain(msg.Author.Username + "#" + msg.Author.Discriminator),
			IconURL:   msg.Author.AvatarURL(),
		})

		if len(entries) >= MaxCompletion {
			break
		}
	}

	return entries
}

func ensureAuthorMapMade(authors *map[discord.UserID]struct{}) {
	if *authors == nil {
		*authors = make(map[discord.UserID]struct{}, MaxCompletion)
	}
}

func Presences(s *state.Instance, word string) []cchat.CompletionEntry {
	presences, err := s.Presences(0)
	if err != nil {
		return nil
	}

	var entries []cchat.CompletionEntry
	var distances map[string]int

	for _, presence := range presences {
		rank := rankFunc(word, presence.User.Username)
		if rank == -1 {
			continue
		}

		ensureEntriesMade(&entries)
		ensureDistancesMade(&distances)

		raw := presence.User.Mention()

		entries = append(entries, cchat.CompletionEntry{
			Raw:       raw,
			Text:      text.Plain(presence.User.Username + "#" + presence.User.Discriminator),
			Secondary: text.Plain(FormatStatus(presence.Status)),
			IconURL:   presence.User.AvatarURL(),
		})

		distances[raw] = rank

		if len(entries) >= MaxCompletion {
			break
		}
	}

	sortDistances(entries, distances)
	return entries
}

func FormatStatus(status discord.Status) string {
	switch status {
	case discord.OnlineStatus:
		return "Online"
	case discord.DoNotDisturbStatus:
		return "Busy"
	case discord.IdleStatus:
		return "Idle"
	case discord.InvisibleStatus:
		return "Invisible"
	case discord.OfflineStatus:
		return "Offline"
	default:
		return strings.Title(string(status))
	}
}

func (ch ChannelCompleter) CompleteMentions(word string) []cchat.CompletionEntry {
	// If there is no input, then we should grab the latest messages.
	if word == "" {
		msgs, _ := ch.State.Store.Messages(ch.ID)
		g, _ := ch.State.Store.Guild(ch.GuildID) // nil is fine

		return GuildMessageMentions(msgs, ch.State, g)
	}

	var entries []cchat.CompletionEntry
	var distances map[string]int

	// If we're not in a guild, then we can check the list of recipients.
	if !ch.GuildID.IsValid() {
		c, err := ch.State.Store.Channel(ch.ID)
		if err != nil {
			return nil
		}

		for _, u := range c.DMRecipients {
			rank := rankFunc(word, u.Username)
			if rank == -1 {
				continue
			}

			ensureEntriesMade(&entries)
			ensureDistancesMade(&distances)

			raw := u.Mention()

			entries = append(entries, cchat.CompletionEntry{
				Raw:       raw,
				Text:      text.Rich{Content: u.Username},
				Secondary: text.Rich{Content: u.Username + "#" + u.Discriminator},
				IconURL:   u.AvatarURL(),
			})

			distances[raw] = rank

			if len(entries) >= MaxCompletion {
				break
			}
		}

		sortDistances(entries, distances)
		return entries
	}

	// If we're in a guild, then we should search for (all) members.
	m, merr := ch.State.Store.Members(ch.GuildID)
	g, gerr := ch.State.Store.Guild(ch.GuildID)

	if merr != nil || gerr != nil {
		return nil
	}

	// If we couldn't find any members, then we can request Discord to
	// search for them.
	if len(m) == 0 {
		ch.State.MemberState.SearchMember(ch.GuildID, word)
		return nil
	}

	for _, mem := range m {
		rank := memberMatchString(word, &mem)
		if rank == -1 {
			continue
		}

		ensureEntriesMade(&entries)
		ensureDistancesMade(&distances)

		raw := mem.User.Mention()

		entries = append(entries, cchat.CompletionEntry{
			Raw:       raw,
			Text:      message.RenderMemberName(mem, *g, ch.State),
			Secondary: text.Plain(mem.User.Username + "#" + mem.User.Discriminator),
			IconURL:   mem.User.AvatarURL(),
		})

		distances[raw] = rank

		if len(entries) >= MaxCompletion {
			break
		}
	}

	sortDistances(entries, distances)
	return entries
}

func memberMatchString(word string, m *discord.Member) int {
	return rankFunc(word, m.User.Username+" "+m.Nick)
}
