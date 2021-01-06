package complete

import (
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/message"
	"github.com/diamondburned/cchat-discord/internal/discord/state"
	"github.com/diamondburned/cchat-discord/internal/segments/mention"
	"github.com/diamondburned/cchat-discord/internal/urlutils"
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

	var entries = make([]cchat.CompletionEntry, 0, MaxCompletion)
	var authors = make(map[discord.UserID]struct{}, MaxCompletion)

	for _, msg := range msgs {
		// If we've already added the author into the list, then skip.
		if _, ok := authors[msg.Author.ID]; ok {
			continue
		}

		authors[msg.Author.ID] = struct{}{}

		user := mention.NewUser(msg.Author)
		user.WithGuildID(msg.GuildID)

		if guild != nil && state != nil {
			user.WithState(state.State)
			user.WithGuild(*guild)
		}

		entries = append(entries, cchat.CompletionEntry{
			Raw:       msg.Author.Mention(),
			Text:      message.RenderAuthorName(user),
			Secondary: text.Plain(msg.Author.Username + "#" + msg.Author.Discriminator),
			IconURL:   msg.Author.AvatarURL(),
		})

		if len(entries) >= MaxCompletion {
			break
		}
	}

	return entries
}

// AllUsers checks for friends and presences.
func AllUsers(s *state.Instance, word string) []cchat.CompletionEntry {
	var full bool

	var friends map[discord.UserID]struct{}
	var entries []cchat.CompletionEntry
	var distances map[string]int

	// Search for friends first.
	s.RelationshipState.Each(func(r *discord.Relationship) bool {
		// Skip blocked users or strangers.
		if r.Type == 0 || r.Type == discord.BlockedRelationship {
			return false
		}

		rank := rankFunc(word, r.User.Username)
		if rank == -1 {
			return false
		}

		if friends == nil {
			friends = map[discord.UserID]struct{}{}
		}

		friends[r.UserID] = struct{}{}

		ensureEntriesMade(&entries)
		ensureDistancesMade(&distances)

		raw := r.User.Mention()

		var status = gateway.UnknownStatus
		if p, _ := s.PresenceStore.Presence(0, r.UserID); p != nil {
			status = p.Status
		}

		entries = append(entries, cchat.CompletionEntry{
			Raw:       raw,
			Text:      text.Plain(r.User.Username + "#" + r.User.Discriminator),
			Secondary: text.Plain(FormatStatus(status) + " - " + FormatRelationshipType(r.Type)),
			IconURL:   urlutils.AvatarURL(r.User.AvatarURL()),
		})

		distances[raw] = rank

		full = len(entries) >= MaxCompletion
		return full
	})

	if full {
		goto Full
	}

	// Search for presences.
	s.PresenceStore.Each(0, func(p *gateway.Presence) bool {
		// Avoid duplicates.
		if _, ok := friends[p.User.ID]; ok {
			return false
		}

		rank := rankFunc(word, p.User.Username)
		if rank == -1 {
			return false
		}

		ensureEntriesMade(&entries)
		ensureDistancesMade(&distances)

		raw := p.User.Mention()

		entries = append(entries, cchat.CompletionEntry{
			Raw:       raw,
			Text:      text.Plain(p.User.Username + "#" + p.User.Discriminator),
			Secondary: text.Plain(FormatStatus(p.Status)),
			IconURL:   urlutils.AvatarURL(p.User.AvatarURL()),
		})

		distances[raw] = rank

		full = len(entries) >= MaxCompletion
		return full
	})

Full:
	sortDistances(entries, distances)
	return entries
}

func FormatStatus(status gateway.Status) string {
	switch status {
	case gateway.OnlineStatus:
		return "Online"
	case gateway.DoNotDisturbStatus:
		return "Busy"
	case gateway.IdleStatus:
		return "Idle"
	case gateway.InvisibleStatus:
		return "Invisible"
	case gateway.OfflineStatus:
		fallthrough
	default:
		return "Offline"
	}
}

func FormatRelationshipType(relaType discord.RelationshipType) string {
	switch relaType {
	case discord.BlockedRelationship:
		return "Blocked"
	case discord.FriendRelationship:
		return "Friend"
	case discord.IncomingFriendRequest:
		return "Incoming friend request"
	case discord.SentFriendRequest:
		return "Friend request sent"
	default:
		return ""
	}
}

func (ch ChannelCompleter) CompleteMentions(word string) []cchat.CompletionEntry {
	// If there is no input, then we should grab the latest messages.
	if word == "" {
		msgs, _ := ch.State.Cabinet.Messages(ch.ID)
		g, _ := ch.State.Cabinet.Guild(ch.GuildID) // nil is fine

		return GuildMessageMentions(msgs, ch.State, g)
	}

	var entries []cchat.CompletionEntry
	var distances map[string]int

	// If we're not in a guild, then we can check the list of recipients.
	if !ch.GuildID.IsValid() {
		c, err := ch.State.Cabinet.Channel(ch.ID)
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
				IconURL:   urlutils.AvatarURL(u.AvatarURL()),
			})

			distances[raw] = rank

			if len(entries) >= MaxCompletion {
				break
			}
		}

		sortDistances(entries, distances)
		return entries
	}

	// Prioritize searching the guild's presences because we don't need to copy
	// slices.
	ch.State.MemberStore.Each(ch.GuildID, func(m *discord.Member) (stop bool) {
		rank := memberMatchString(word, m)
		if rank == -1 {
			return false
		}

		ensureEntriesMade(&entries)
		ensureDistancesMade(&distances)

		user := mention.NewUser(m.User)
		user.WithGuildID(ch.GuildID)
		user.WithMember(*m)
		user.WithState(ch.State.State)

		raw := m.User.Mention()

		entries = append(entries, cchat.CompletionEntry{
			Raw:       raw,
			Text:      message.RenderAuthorName(user),
			Secondary: text.Plain(m.User.Username + "#" + m.User.Discriminator),
			IconURL:   user.Avatar(),
		})

		distances[raw] = rank

		return len(entries) >= MaxCompletion
	})

	sortDistances(entries, distances)
	return entries
}

func memberMatchString(word string, m *discord.Member) int {
	return rankFunc(word, m.User.Username+" "+m.Nick)
}
