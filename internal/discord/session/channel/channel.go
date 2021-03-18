package channel

import (
	"context"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/session/channel/messenger"
	"github.com/diamondburned/cchat-discord/internal/discord/session/channel/shared"
	"github.com/diamondburned/cchat-discord/internal/discord/state"
	"github.com/diamondburned/cchat/utils/empty"
	"github.com/pkg/errors"
)

type Channel struct {
	empty.Server
	shared.Channel
	commander cchat.Commander
}

var _ cchat.Server = (*Channel)(nil)

func New(s *state.Instance, ch discord.Channel) (cchat.Server, error) {
	channel, err := NewChannel(s, ch)
	if err != nil {
		return nil, err
	}
	return channel, nil
}

func NewChannel(s *state.Instance, ch discord.Channel) (Channel, error) {
	// Ensure the state keeps the channel's permission.
	if ch.GuildID.IsValid() {
		_, err := s.Permissions(ch.ID, s.UserID)
		if err != nil {
			return Channel{}, errors.Wrap(err, "failed to get permission")
		}
	}

	sharedCh := shared.Channel{
		ID:      ch.ID,
		GuildID: ch.GuildID,
		State:   s,
	}

	return Channel{
		Channel:   sharedCh,
		commander: NewCommander(sharedCh),
	}, nil
}

func (ch Channel) ID() cchat.ID {
	return ch.Channel.ID.String()
}

func (ch Channel) Name(_ context.Context, l cchat.LabelContainer) (func(), error) {
	return ch.State.Labels.AddChannelLabel(ch.Channel.ID, l), nil
}

func (ch Channel) Columnate() bool { return false }

func (ch Channel) AsCommander() cchat.Commander {
	return ch.commander
}

func (ch Channel) AsMessenger() cchat.Messenger {
	if !ch.HasPermission(discord.PermissionViewChannel) {
		return nil
	}

	return messenger.New(ch.Channel)
}
