package typer

import (
	"time"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/message"
	"github.com/diamondburned/cchat-discord/internal/discord/state"
	"github.com/diamondburned/cchat-discord/internal/segments/mention"
	"github.com/pkg/errors"
)

type Typer struct {
	message.Author
	time discord.UnixTimestamp
}

var _ cchat.Typer = (*Typer)(nil)

// New creates a new Typer that satisfies cchat.Typer.
func New(s *state.Instance, ev *gateway.TypingStartEvent) (*Typer, error) {
	var user *mention.User

	if ev.GuildID.IsValid() {
		if ev.Member == nil {
			m, err := s.Cabinet.Member(ev.GuildID, ev.UserID)
			if err != nil {
				// There's no other way we could get the user (other than to
				// check for presences), so we bail.
				return nil, errors.Wrap(err, "failed to get member")
			}
			ev.Member = m
		}

		user = mention.NewUser(ev.Member.User)
		user.WithMember(*ev.Member)
	} else {
		c, err := s.Cabinet.Channel(ev.ChannelID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get channel")
		}

		for _, recipient := range c.DMRecipients {
			if recipient.ID != ev.UserID {
				continue
			}

			user = mention.NewUser(recipient)
			break
		}
	}

	user.WithGuildID(ev.GuildID)
	user.WithState(s.State)
	user.Prefetch()

	return &Typer{
		Author: message.NewAuthor(user),
		time:   ev.Timestamp,
	}, nil
}

func (t Typer) Time() time.Time {
	return t.time.Time()
}
