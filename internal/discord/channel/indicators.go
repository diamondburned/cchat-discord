package channel

import (
	"time"

	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/channel/typer"
	"github.com/diamondburned/ningen/states/read"
	"github.com/pkg/errors"
)

var (
	_ cchat.TypingIndicator = (*Channel)(nil)
	_ cchat.UnreadIndicator = (*Channel)(nil)
)

// IsTypingIndicator returns true.
func (ch *Channel) IsTypingIndicator() bool { return true }

func (ch *Channel) Typing() error {
	return ch.state.Typing(ch.id)
}

// TypingTimeout returns 10 seconds.
func (ch *Channel) TypingTimeout() time.Duration {
	return 10 * time.Second
}

func (ch *Channel) TypingSubscribe(ti cchat.TypingContainer) (func(), error) {
	return ch.state.AddHandler(func(t *gateway.TypingStartEvent) {
		// Ignore channel mismatch or if the typing event is ours.
		if t.ChannelID != ch.id || t.UserID == ch.state.UserID {
			return
		}
		if typer, err := typer.New(ch.state, t); err == nil {
			ti.AddTyper(typer)
		}
	}), nil
}

// muted returns if this channel is muted. This includes the channel's category
// and guild.
func (ch *Channel) muted() bool {
	return (ch.guildID.IsValid() && ch.state.MutedState.Guild(ch.guildID, false)) ||
		ch.state.MutedState.Channel(ch.id) ||
		ch.state.MutedState.Category(ch.id)
}

// IsUnreadIndicator returns true.
func (ch *Channel) IsUnreadIndicator() bool { return true }

func (ch *Channel) UnreadIndicate(indicator cchat.UnreadContainer) (func(), error) {
	if rs := ch.state.ReadState.FindLast(ch.id); rs != nil {
		c, err := ch.self()
		if err != nil {
			return nil, errors.Wrap(err, "Failed to get self channel")
		}

		if c.LastMessageID > rs.LastMessageID && !ch.muted() {
			indicator.SetUnread(true, rs.MentionCount > 0)
		}
	}

	return ch.state.ReadState.OnUpdate(func(ev *read.UpdateEvent) {
		if ch.id == ev.ChannelID && !ch.muted() {
			indicator.SetUnread(ev.Unread, ev.MentionCount > 0)
		}
	}), nil
}
