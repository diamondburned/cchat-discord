// Package shared contains channel utilities.
package shared

import (
	"errors"
	"strings"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/cchat-discord/internal/discord/state"
)

// ChannelName returns the channel name if any, otherwise it formats its own
// name into a list of recipients.
func ChannelName(ch discord.Channel) string {
	switch ch.Type {
	case discord.DirectMessage, discord.GroupDM:
		if len(ch.DMRecipients) > 0 {
			return FormatRecipients(ch.DMRecipients)
		}

	default:
		if ch.Name == "" {
			break
		}

		if ch.NSFW {
			return "#" + ch.Name + " (nsfw)"
		} else {
			return "#" + ch.Name
		}
	}

	return ch.ID.String()
}

// FormatRecipients joins the given list of users into a string listing all
// recipients with English punctuation rules.
func FormatRecipients(users []discord.User) string {
	switch len(users) {
	case 0:
		return ""
	case 1:
		return users[0].Username
	case 2:
		return users[0].Username + " and " + users[1].Username
	}

	var usernames = make([]string, len(users)-1)
	for i, user := range users[:len(users)-1] {
		usernames[i] = user.Username
	}

	return strings.Join(usernames, ", ") + " and " + users[len(users)-1].Username
}

type Channel struct {
	ID      discord.ChannelID
	GuildID discord.GuildID
	State   *state.Instance
}

// HasPermission returns true if the current user has the given permissions in
// the channel.
func (ch Channel) HasPermission(perms ...discord.Permissions) bool {
	// Assume we have permissions in a direct message channel.
	if !ch.GuildID.IsValid() {
		return true
	}

	p, err := ch.State.StateOnly().Permissions(ch.ID, ch.State.UserID)
	if err != nil {
		return false
	}

	for _, perm := range perms {
		if !p.Has(perm) {
			return false
		}
	}

	return true
}

func (ch Channel) Messages() ([]discord.Message, error) {
	return ch.State.Cabinet.Messages(ch.ID)
}

func (ch Channel) Guild() (*discord.Guild, error) {
	if !ch.GuildID.IsValid() {
		return nil, errors.New("channel not in guild")
	}
	return ch.State.Cabinet.Guild(ch.GuildID)
}

func (ch Channel) Self() (*discord.Channel, error) {
	return ch.State.Cabinet.Channel(ch.ID)
}
