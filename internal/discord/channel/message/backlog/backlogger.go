package backlog

import (
	"context"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/channel/shared"
	"github.com/diamondburned/cchat-discord/internal/discord/message"
	"github.com/pkg/errors"
)

type Backlogger struct {
	shared.Channel
}

func New(ch shared.Channel) cchat.Backlogger {
	return Backlogger{ch}
}

func (bl Backlogger) Backlog(ctx context.Context, b cchat.ID, c cchat.MessagesContainer) error {
	p, err := discord.ParseSnowflake(b)
	if err != nil {
		return errors.Wrap(err, "Failed to parse snowflake")
	}

	s := bl.State.WithContext(ctx)

	m, err := s.MessagesBefore(bl.ID, discord.MessageID(p), uint(bl.State.MaxMessages()))
	if err != nil {
		return errors.Wrap(err, "Failed to get messages")
	}

	// Create the backlog without any member information.
	g, err := s.Guild(bl.GuildID)
	if err != nil {
		return errors.Wrap(err, "Failed to get guild")
	}

	for _, m := range m {
		// Discord sucks.
		m.GuildID = bl.GuildID

		c.CreateMessage(message.NewBacklogMessage(m, bl.State, *g))
	}

	return nil
}
