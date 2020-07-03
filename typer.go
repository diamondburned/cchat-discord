package discord

import (
	"time"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/arikawa/state"
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

func NewTyper(store state.Store, ev *gateway.TypingStartEvent) (*Typer, error) {
	if ev.GuildID.Valid() {
		g, err := store.Guild(ev.GuildID)
		if err != nil {
			return nil, err
		}

		if ev.Member == nil {
			ev.Member, err = store.Member(ev.GuildID, ev.UserID)
			if err != nil {
				return nil, err
			}
		}

		return &Typer{
			Author: NewGuildMember(*ev.Member, *g),
			time:   ev.Timestamp,
		}, nil
	}

	c, err := store.Channel(ev.ChannelID)
	if err != nil {
		return nil, err
	}

	for _, user := range c.DMRecipients {
		if user.ID == ev.UserID {
			return &Typer{
				Author: NewUser(user),
				time:   ev.Timestamp,
			}, nil
		}
	}

	return nil, errors.New("typer not found in state")
}

func (t Typer) Time() time.Time {
	return t.time.Time()
}
