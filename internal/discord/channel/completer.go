package channel

import (
	"strings"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/message"
	"github.com/diamondburned/cchat-discord/internal/discord/state"
	"github.com/diamondburned/cchat-discord/internal/urlutils"
	"github.com/diamondburned/cchat/text"
)

const MaxCompletion = 15

var _ cchat.MessageCompleter = (*Channel)(nil)

// IsMessageCompleter returns true if the user can send messages in this
// channel.
func (ch *Channel) IsMessageCompleter() bool {
	p, err := ch.state.StateOnly().Permissions(ch.id, ch.state.UserID)
	if err != nil {
		return false
	}

	return p.Has(discord.PermissionSendMessages)
}

// CompleteMessage implements message input completion capability for Discord.
// This method supports user mentions, channel mentions and emojis.
//
// For the individual implementations, refer to channel_completion.go.
func (ch *Channel) CompleteMessage(words []string, i int) (entries []cchat.CompletionEntry) {
	var word = words[i]
	// Word should have at least a character for the char check.
	if len(word) < 1 {
		return
	}

	switch word[0] {
	case '@':
		return ch.completeMentions(word[1:])
	case '#':
		return ch.completeChannels(word[1:])
	case ':':
		return ch.completeEmojis(word[1:])
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

func (ch *Channel) completeMentions(word string) (entries []cchat.CompletionEntry) {
	// If there is no input, then we should grab the latest messages.
	if word == "" {
		msgs, _ := ch.messages()
		g, _ := ch.guild() // nil is fine

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
			entries = append(entries, completionUser(ch.state, msg.Author, g))

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
	if !ch.guildID.IsValid() {
		c, err := ch.self()
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
	m, merr := ch.state.Store.Members(ch.guildID)
	g, gerr := ch.guild()

	if merr != nil || gerr != nil {
		return
	}

	// If we couldn't find any members, then we can request Discord to
	// search for them.
	if len(m) == 0 {
		ch.state.MemberState.SearchMember(ch.guildID, word)
		return
	}

	for _, mem := range m {
		if contains(match, mem.User.Username, mem.Nick) {
			entries = append(entries, cchat.CompletionEntry{
				Raw:       mem.User.Mention(),
				Text:      message.RenderMemberName(mem, *g, ch.state),
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

func (ch *Channel) completeChannels(word string) (entries []cchat.CompletionEntry) {
	// Ignore if empty word.
	if word == "" {
		return
	}

	// Ignore if we're not in a guild.
	if !ch.guildID.IsValid() {
		return
	}

	c, err := ch.state.State.Channels(ch.guildID)
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
			if c, _ := ch.state.Store.Channel(channel.CategoryID); c != nil {
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

func (ch *Channel) completeEmojis(word string) (entries []cchat.CompletionEntry) {
	// Ignore if empty word.
	if word == "" {
		return
	}

	e, err := ch.state.EmojiState.Get(ch.guildID)
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

func contains(contains string, strs ...string) bool {
	for _, str := range strs {
		if strings.Contains(strings.ToLower(str), contains) {
			return true
		}
	}

	return false
}
