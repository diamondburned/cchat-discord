package channel

import (
	"context"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/channel/shared"
	"github.com/diamondburned/cchat-discord/internal/discord/state"
	"github.com/diamondburned/cchat-discord/internal/urlutils"
	"github.com/diamondburned/cchat/text"
	"github.com/pkg/errors"
)

type Private struct {
	Channel
}

var _ cchat.Server = (*Private)(nil)

func NewPrivate(s *state.Instance, ch discord.Channel) (cchat.Server, error) {
	if ch.GuildID.IsValid() {
		return nil, errors.New("channel has valid guild ID: not a DM")
	}

	channel, err := NewChannel(s, ch)
	if err != nil {
		return nil, err
	}

	return Private{Channel: channel}, nil
}

func (priv Private) Name() text.Rich {
	c, err := priv.Self()
	if err != nil {
		return text.Rich{Content: priv.ID()}
	}

	return text.Plain(shared.PrivateName(*c))
}

func (priv Private) AsIconer() cchat.Iconer {
	return NewAvatarIcon(priv.State)
}

type AvatarIcon struct {
	State *state.Instance
}

func NewAvatarIcon(state *state.Instance) cchat.Iconer {
	return AvatarIcon{state}
}

func (avy AvatarIcon) Icon(ctx context.Context, iconer cchat.IconContainer) (func(), error) {
	u, err := avy.State.WithContext(ctx).Me()
	if err != nil {
		// This shouldn't happen.
		return nil, errors.Wrap(err, "Failed to get guild")
	}

	// Used for comparison.
	if u.Avatar != "" {
		iconer.SetIcon(urlutils.AvatarURL(u.AvatarURL()))
	}

	selfID := u.ID

	return avy.State.AddHandler(func(update *gateway.UserUpdateEvent) {
		if selfID == update.ID {
			iconer.SetIcon(urlutils.AvatarURL(update.AvatarURL()))
		}
	}), nil
}
