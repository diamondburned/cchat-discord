// Package shared contains channel utilities.
package shared

import (
	"errors"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/cchat-discord/internal/discord/state"
)

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

	p, err := ch.State.Permissions(ch.ID, ch.State.UserID)
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
