package channel

import (
	"context"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/channel/message"
	"github.com/diamondburned/cchat-discord/internal/discord/channel/shared"
	"github.com/diamondburned/cchat-discord/internal/discord/state"
	"github.com/diamondburned/cchat-discord/internal/urlutils"
	"github.com/diamondburned/cchat/text"
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

func (ch Channel) Name() text.Rich {
	c, err := ch.Self()
	if err != nil {
		return text.Rich{Content: ch.ID()}
	}

	return text.Plain(shared.ChannelName(*c))
}

func (ch Channel) AsCommander() cchat.Commander {
	return ch.commander
}

func (ch Channel) AsMessenger() cchat.Messenger {
	if !ch.HasPermission(discord.PermissionViewChannel) {
		return nil
	}

	return message.New(ch.Channel)
}

func (ch Channel) AsIconer() cchat.Iconer {
	// Guild channels never have an icon.
	if ch.GuildID.IsValid() {
		return nil
	}

	c, err := ch.Self()
	if err != nil {
		return nil
	}

	// Only DM channels should have an icon.
	if c.Type != discord.DirectMessage {
		return nil
	}

	return PresenceAvatar{
		user:  c.DMRecipients[0],
		guild: ch.GuildID,
		state: ch.State,
	}
}

type PresenceAvatar struct {
	user  discord.User
	guild discord.GuildID
	state *state.Instance
}

func (avy PresenceAvatar) Icon(ctx context.Context, iconer cchat.IconContainer) (func(), error) {
	if avy.user.Avatar != "" {
		iconer.SetIcon(urlutils.AvatarURL(avy.user.AvatarURL()))
	}

	// There are so many other places that could be checked, but this is good
	// enough.

	c, err := avy.state.Presence(avy.guild, avy.user.ID)
	if err == nil && c.User.Avatar != "" {
		iconer.SetIcon(urlutils.AvatarURL(c.User.AvatarURL()))
	}

	return avy.state.AddHandler(func(update *gateway.PresenceUpdateEvent) {
		if avy.user.ID == update.User.ID {
			iconer.SetIcon(urlutils.AvatarURL(update.User.AvatarURL()))
		}
	}), nil
}
