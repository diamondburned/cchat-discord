package channel

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/channel/message"
	"github.com/diamondburned/cchat-discord/internal/discord/channel/shared"
	"github.com/diamondburned/cchat-discord/internal/discord/state"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/cchat/utils/empty"
	"github.com/pkg/errors"
)

type Channel struct {
	empty.Server

	*shared.Channel
	commander cchat.Commander
}

var _ cchat.Server = (*Channel)(nil)

func New(s *state.Instance, ch discord.Channel) (cchat.Server, error) {
	// Ensure the state keeps the channel's permission.
	_, err := s.Permissions(ch.ID, s.UserID)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get permission")
	}

	sharedCh := &shared.Channel{
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
		return text.Rich{Content: ch.Channel.ID.String()}
	}

	if c.NSFW {
		return text.Rich{Content: "#!" + c.Name}
	} else {
		return text.Rich{Content: "#" + c.Name}
	}
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
