package discord

import (
	"time"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/cchat"
	"github.com/pkg/errors"
)

type Typer struct {
	Author
	time discord.UnixTimestamp
}

var _ cchat.Typer = (*Typer)(nil)

func NewTyperAuthor(author Author, ev *gateway.TypingStartEvent) Typer {
	return Typer{
		Author: author,
		time:   ev.Timestamp,
	}
}

func NewTyper(s *Session, ev *gateway.TypingStartEvent) (*Typer, error) {
	if ev.GuildID.Valid() {
		g, err := s.Store.Guild(ev.GuildID)
		if err != nil {
			return nil, err
		}

		if ev.Member == nil {
			ev.Member, err = s.Store.Member(ev.GuildID, ev.UserID)
			if err != nil {
				return nil, err
			}
		}

		return &Typer{
			Author: NewGuildMember(*ev.Member, *g, s),
			time:   ev.Timestamp,
		}, nil
	}

	c, err := s.Store.Channel(ev.ChannelID)
	if err != nil {
		return nil, err
	}

	for _, user := range c.DMRecipients {
		if user.ID == ev.UserID {
			return &Typer{
				Author: NewUser(user, s),
				time:   ev.Timestamp,
			}, nil
		}
	}

	return nil, errors.New("typer not found in state")
}

func (t Typer) Time() time.Time {
	return t.time.Time()
}
