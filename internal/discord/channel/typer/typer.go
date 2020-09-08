package typer

import (
	"errors"
	"time"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/message"
	"github.com/diamondburned/cchat-discord/internal/discord/state"
)

type Typer struct {
	message.Author
	time discord.UnixTimestamp
}

var _ cchat.Typer = (*Typer)(nil)

func NewFromAuthor(author message.Author, ev *gateway.TypingStartEvent) Typer {
	return Typer{
		Author: author,
		time:   ev.Timestamp,
	}
}

func New(s *state.Instance, ev *gateway.TypingStartEvent) (*Typer, error) {
	if ev.GuildID.IsValid() {
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
			Author: message.NewGuildMember(*ev.Member, *g, s),
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
				Author: message.NewUser(user, s),
				time:   ev.Timestamp,
			}, nil
		}
	}

	return nil, errors.New("typer not found in state")
}

func (t Typer) Time() time.Time {
	return t.time.Time()
}
