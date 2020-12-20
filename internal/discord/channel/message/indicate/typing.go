package indicate

import (
	"time"

	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/channel/shared"
	"github.com/diamondburned/cchat-discord/internal/discord/channel/typer"
)

type TypingIndicator struct {
	shared.Channel
}

func NewTyping(ch shared.Channel) cchat.TypingIndicator {
	return TypingIndicator{ch}
}

func (ti TypingIndicator) Typing() error {
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
		if typer, err := typer.New(ti.State, t); err == nil {
			tc.AddTyper(typer)
		}
	}), nil
}
