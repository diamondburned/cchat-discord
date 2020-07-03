package discord

import (
	"time"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/arikawa/state"
	"github.com/diamondburned/cchat"
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
	if t.ChannelID != ch.id {
		return
	}

	if ch.guildID.Valid() {
		g, err := ch.session.Store.Guild(t.GuildID)
		if err != nil {
			return
		}

		if t.Member == nil {
			t.Member, err = ch.session.Store.Member(t.GuildID, t.UserID)
			if err != nil {
				return
			}
		}

		ti.AddTyper(NewTyper(NewGuildMember(*t.Member, *g), t))
		return
	}

	c, err := ch.self()
	if err != nil {
		return
	}

	for _, user := range c.DMRecipients {
		if user.ID == t.UserID {
			ti.AddTyper(NewTyper(NewUser(user), t))
			return
		}
	}
}

func (t Typer) Time() time.Time {
	return t.time.Time()
}
