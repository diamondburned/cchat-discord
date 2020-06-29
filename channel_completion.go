package discord

import (
	"strings"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/state"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat/text"
)

const MaxCompletion = 15

func completionUserEntry(s state.Store, u discord.User, g *discord.Guild) cchat.CompletionEntry {
	if g != nil {
		m, err := s.Member(g.ID, u.ID)
		if err == nil {
			return cchat.CompletionEntry{
				Raw:     u.Mention(),
				Text:    RenderMemberName(*m, *g),
				IconURL: u.AvatarURL(),
			}
		}
	}

	return cchat.CompletionEntry{
		Raw:     u.Mention(),
		Text:    text.Rich{Content: u.Username},
		IconURL: u.AvatarURL(),
	}
}

func (ch *Channel) completeMentions(word string) (entries []cchat.CompletionEntry) {
	// If there is no input, then we should grab the latest messages.
	if word == "" {
		msgs, _ := ch.messages()
		g, _ := ch.guild() // nil is fine

		// Keep track of the number of authors.
		// TODO: fix excess allocations
		var authors = make(map[discord.Snowflake]struct{}, MaxCompletion)

		for _, msg := range msgs {
			// If we've already added the author into the list, then skip.
			if _, ok := authors[msg.Author.ID]; ok {
				continue
			}

			// Record the current author and add the entry to the list.
			authors[msg.Author.ID] = struct{}{}
			entries = append(entries, completionUserEntry(ch.session.Store, msg.Author, g))

			if len(entries) >= MaxCompletion {
				return
			}
		}

		return
	}

	// Lower-case everything for a case-insensitive match. startsWith() should
	// do the rest.
	var match = strings.ToLower(word)

	// If we're not in a guild, then we can check the list of recipients.
	if !ch.guildID.Valid() {
		c, err := ch.self()
		if err == nil {
			for _, u := range c.DMRecipients {
				if startsWith(match, u.Username) {
					entries = append(entries, cchat.CompletionEntry{
						Raw:     u.Mention(),
						Text:    text.Rich{Content: u.Username},
						IconURL: u.AvatarURL(),
					})
					if len(entries) >= MaxCompletion {
						return
					}
				}
			}
		}

		return
	}

	// If we're in a guild, then we should search for (all) members.
	m, merr := ch.session.Store.Members(ch.guildID)
	g, gerr := ch.guild()

	if merr != nil || gerr != nil {
		return
	}

	// If we couldn't find any members, then we can request Discord to
	// search for them.
	if len(m) == 0 {
		ch.session.Members.SearchMember(ch.guildID, word)
		return
	}

	for _, mem := range m {
		if startsWith(match, mem.User.Username, mem.Nick) {
			entries = append(entries, cchat.CompletionEntry{
				Raw:     mem.User.Mention(),
				Text:    RenderMemberName(mem, *g),
				IconURL: mem.User.AvatarURL(),
			})
			if len(entries) >= MaxCompletion {
				return
			}
		}
	}

	return
}

func (ch *Channel) completeChannels(word string) (entries []cchat.CompletionEntry) {
	// Ignore if empty word.
	if word == "" {
		return
	}

	// Ignore if we're not in a guild.
	if !ch.guildID.Valid() {
		return
	}

	c, err := ch.session.State.Channels(ch.guildID)
	if err != nil {
		return
	}

	var match = strings.ToLower(word)

	for _, ch := range c {
		if startsWith(match, ch.Name) {
			entries = append(entries, cchat.CompletionEntry{
				Raw:  ch.Mention(),
				Text: text.Rich{Content: "#" + ch.Name},
			})
			if len(entries) >= MaxCompletion {
				return
			}
		}
	}

	return
}

func (ch *Channel) completeEmojis(word string) (entries []cchat.CompletionEntry) {
	// Ignore if empty word.
	if word == "" {
		return
	}

	e, err := ch.session.Emoji.Get(ch.guildID)
	if err != nil {
		return
	}

	var match = strings.ToLower(word)

	for _, guild := range e {
		for _, emoji := range guild.Emojis {
			if startsWith(match, emoji.Name) {
				entries = append(entries, cchat.CompletionEntry{
					Raw:     emoji.String(),
					Text:    text.Rich{Content: ":" + emoji.Name + ":"},
					IconURL: emoji.EmojiURL() + "?size=32", // small
				})
				if len(entries) >= MaxCompletion {
					return
				}
			}
		}
	}

	return
}

func startsWith(contains string, strs ...string) bool {
	for _, str := range strs {
		if strings.HasPrefix(strings.ToLower(str), contains) {
			return true
		}
	}

	return false
}
