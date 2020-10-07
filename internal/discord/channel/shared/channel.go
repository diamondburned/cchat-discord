package shared

import (
	"github.com/diamondburned/arikawa/discord"
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
	return ch.State.Store.Guild(ch.GuildID)
}

func (ch Channel) Self() (*discord.Channel, error) {
	return ch.State.Store.Channel(ch.ID)
}
