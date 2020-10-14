package complete

import (
	"strings"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/message"
	"github.com/diamondburned/cchat-discord/internal/discord/state"
	"github.com/diamondburned/cchat/text"
)

func (ch Completer) CompleteMentions(word string) (entries []cchat.CompletionEntry) {
	// If there is no input, then we should grab the latest messages.
	if word == "" {
		msgs, _ := ch.State.Store.Messages(ch.ID)
		g, _ := ch.State.Store.Guild(ch.GuildID) // nil is fine

		// Keep track of the number of authors.
		// TODO: fix excess allocations
		var authors = make(map[discord.UserID]struct{}, MaxCompletion)

		for _, msg := range msgs {
			// If we've already added the author into the list, then skip.
			if _, ok := authors[msg.Author.ID]; ok {
				continue
			}

			// Record the current author and add the entry to the list.
			authors[msg.Author.ID] = struct{}{}
			entries = append(entries, completionUser(ch.State, msg.Author, g))

			if len(entries) >= MaxCompletion {
				return
			}
		}

		return
	}

	// Lower-case everything for a case-insensitive match. contains() should
	// do the rest.
	var match = strings.ToLower(word)

	// If we're not in a guild, then we can check the list of recipients.
	if !ch.GuildID.IsValid() {
		c, err := ch.State.Store.Channel(ch.ID)
		if err != nil {
			return
		}

		for _, u := range c.DMRecipients {
			if contains(match, u.Username) {
				entries = append(entries, cchat.CompletionEntry{
					Raw:       u.Mention(),
					Text:      text.Rich{Content: u.Username},
					Secondary: text.Rich{Content: u.Username + "#" + u.Discriminator},
					IconURL:   u.AvatarURL(),
				})
				if len(entries) >= MaxCompletion {
					return
				}
			}
		}

		return
	}

	// If we're in a guild, then we should search for (all) members.
	m, merr := ch.State.Store.Members(ch.GuildID)
	g, gerr := ch.State.Store.Guild(ch.GuildID)

	if merr != nil || gerr != nil {
		return
	}

	// If we couldn't find any members, then we can request Discord to
	// search for them.
	if len(m) == 0 {
		ch.State.MemberState.SearchMember(ch.GuildID, word)
		return
	}

	for _, mem := range m {
		if contains(match, mem.User.Username, mem.Nick) {
			entries = append(entries, cchat.CompletionEntry{
				Raw:       mem.User.Mention(),
				Text:      message.RenderMemberName(mem, *g, ch.State),
				Secondary: text.Rich{Content: mem.User.Username + "#" + mem.User.Discriminator},
				IconURL:   mem.User.AvatarURL(),
			})
			if len(entries) >= MaxCompletion {
				return
			}
		}
	}

	return
}

func completionUser(s *state.Instance, u discord.User, g *discord.Guild) cchat.CompletionEntry {
	if g != nil {
		m, err := s.Store.Member(g.ID, u.ID)
		if err == nil {
			return cchat.CompletionEntry{
				Raw:       u.Mention(),
				Text:      message.RenderMemberName(*m, *g, s),
				Secondary: text.Rich{Content: u.Username + "#" + u.Discriminator},
				IconURL:   u.AvatarURL(),
			}
		}
	}

	return cchat.CompletionEntry{
		Raw:       u.Mention(),
		Text:      text.Rich{Content: u.Username},
		Secondary: text.Rich{Content: u.Username + "#" + u.Discriminator},
		IconURL:   u.AvatarURL(),
	}
}
