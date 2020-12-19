// Package shared contains channel utilities.
package shared

import (
	"errors"
	"strings"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/cchat-discord/internal/discord/state"
)

// PrivateName returns the channel name if any, otherwise it formats its own
// name into a list of recipients.
func PrivateName(privCh discord.Channel) string {
	if privCh.Name != "" {
		return privCh.Name
	}

	return FormatRecipients(privCh.DMRecipients)
}

// FormatRecipients joins the given list of users into a string listing all
// recipients with English punctuation rules.
func FormatRecipients(users []discord.User) string {
	switch len(users) {
	case 0:
		return "<Nobody>"
	case 1:
		return users[0].Username
	case 2:
		return users[0].Username + " and " + users[1].Username
	}

	var usernames = make([]string, len(users))
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
	return ch.State.Store.Messages(ch.ID)
}

func (ch Channel) Guild() (*discord.Guild, error) {
	if !ch.GuildID.IsValid() {
		return nil, errors.New("channel not in guild")
	}
	return ch.State.Store.Guild(ch.GuildID)
}

func (ch Channel) Self() (*discord.Channel, error) {
	return ch.State.Store.Channel(ch.ID)
}
