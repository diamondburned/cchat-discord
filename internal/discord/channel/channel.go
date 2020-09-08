package channel

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/state"
	"github.com/diamondburned/cchat/text"
	"github.com/pkg/errors"
)

type Channel struct {
	id      discord.ChannelID
	guildID discord.GuildID
	state   *state.Instance
}

var _ cchat.Server = (*Channel)(nil)

func New(s *state.Instance, ch discord.Channel) (cchat.Server, error) {
	// Ensure the state keeps the channel's permission.
	_, err := s.Permissions(ch.ID, s.UserID)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get permission")
	}

	return &Channel{
		id:      ch.ID,
		guildID: ch.GuildID,
		state:   s,
	}, nil
}

// self does not do IO.
func (ch *Channel) self() (*discord.Channel, error) {
	return ch.state.Store.Channel(ch.id)
}

// messages does not do IO.
func (ch *Channel) messages() ([]discord.Message, error) {
	return ch.state.Store.Messages(ch.id)
}

func (ch *Channel) guild() (*discord.Guild, error) {
	if ch.guildID.IsValid() {
		return ch.state.Store.Guild(ch.guildID)
	}
	return nil, errors.New("channel not in a guild")
}

func (ch *Channel) ID() cchat.ID {
	return ch.id.String()
}

func (ch *Channel) Name() text.Rich {
	c, err := ch.self()
	if err != nil {
		return text.Rich{Content: ch.id.String()}
	}

	if c.NSFW {
		return text.Rich{Content: "#!" + c.Name}
	} else {
		return text.Rich{Content: "#" + c.Name}
	}
}
