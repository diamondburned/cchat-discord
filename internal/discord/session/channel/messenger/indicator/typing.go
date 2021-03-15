package indicator

import (
	"time"

	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/config"
	"github.com/diamondburned/cchat-discord/internal/discord/message"
	"github.com/diamondburned/cchat-discord/internal/discord/session/channel/shared"
	"github.com/diamondburned/cchat-discord/internal/discord/state"
	"github.com/diamondburned/cchat-discord/internal/segments/mention"
	"github.com/pkg/errors"
)

type TypingIndicator struct {
	shared.Channel
}

func NewTyping(ch shared.Channel) cchat.TypingIndicator {
	return TypingIndicator{ch}
}

func (ti TypingIndicator) Typing() error {
	if !config.BroadcastTyping() {
		return nil
	}

	return ti.State.Typing(ti.ID)
}

// TypingTimeout returns 10 seconds.
func (ti TypingIndicator) TypingTimeout() time.Duration {
	return 10 * time.Second
}

func (ti TypingIndicator) TypingSubscribe(tc cchat.TypingContainer) (func(), error) {
	return ti.State.AddHandler(func(t *gateway.TypingStartEvent) {
		// Ignore channel mismatch or if the typing event is ours.
		if t.ChannelID != ti.ID || t.UserID == ti.State.UserID {
			return
		}
		if typer, err := NewTyperUser(ti.State, t); err == nil {
			tc.AddTyper(typer)
		}
	}), nil
}

// New creates a new Typer that satisfies cchat.Typer.
func NewTyperUser(s *state.Instance, ev *gateway.TypingStartEvent) (cchat.User, error) {
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

	return message.NewAuthor(s, user), nil
}
