package channel

import (
	"context"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/message"
	"github.com/pkg/errors"
)

var _ cchat.Backlogger = (*Channel)(nil)

// IsBacklogger returns true if the current user can read the channel's message
// history.
func (ch *Channel) IsBacklogger() bool {
	p, err := ch.state.StateOnly().Permissions(ch.id, ch.state.UserID)
	if err != nil {
		return false
	}

	return p.Has(discord.PermissionViewChannel) && p.Has(discord.PermissionReadMessageHistory)
}

func (ch *Channel) MessagesBefore(ctx context.Context, b cchat.ID, c cchat.MessagePrepender) error {
	p, err := discord.ParseSnowflake(b)
	if err != nil {
		return errors.Wrap(err, "Failed to parse snowflake")
	}

	s := ch.state.WithContext(ctx)

	m, err := s.MessagesBefore(ch.id, discord.MessageID(p), uint(ch.state.MaxMessages()))
	if err != nil {
		return errors.Wrap(err, "Failed to get messages")
	}

	// Create the backlog without any member information.
	g, err := s.Guild(ch.guildID)
	if err != nil {
		return errors.Wrap(err, "Failed to get guild")
	}

	for _, m := range m {
		// Discord sucks.
		m.GuildID = ch.guildID

		c.PrependMessage(message.NewBacklogMessage(m, ch.state, *g))
	}

	return nil
}
