package nickname

import (
	"context"

	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/channel/shared"
	"github.com/diamondburned/cchat-discord/internal/segments/colored"
	"github.com/diamondburned/cchat/text"
	"github.com/pkg/errors"
)

type Nicknamer struct {
	*shared.Channel
}

func New(ch *shared.Channel) cchat.Nicknamer {
	return Nicknamer{ch}
}

func (nn Nicknamer) Nickname(ctx context.Context, labeler cchat.LabelContainer) (func(), error) {
	// We don't have a nickname if we're not in a guild.
	if !nn.GuildID.IsValid() {
		return func() {}, nil
	}

	state := nn.State.WithContext(ctx)

	// MemberColor should fill up the state cache.
	c, err := state.MemberColor(nn.GuildID, nn.State.UserID)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get self member color")
	}

	m, err := state.Member(nn.GuildID, nn.State.UserID)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get self member")
	}

	var rich = text.Rich{Content: m.User.Username}
	if m.Nick != "" {
		rich.Content = m.Nick
	}
	if c > 0 {
		rich.Segments = []text.Segment{
			colored.New(len(rich.Content), c.Uint32()),
		}
	}

	labeler.SetLabel(rich)

	// Copy the user ID to use.
	var selfID = m.User.ID

	return nn.State.AddHandler(func(g *gateway.GuildMemberUpdateEvent) {
		if g.GuildID != nn.GuildID || g.User.ID != selfID {
			return
		}

		var rich = text.Rich{Content: m.User.Username}
		if m.Nick != "" {
			rich.Content = m.Nick
		}

		c, err := nn.State.MemberColor(g.GuildID, selfID)
		if err == nil {
			rich.Segments = []text.Segment{
				colored.New(len(rich.Content), c.Uint32()),
			}
		}

		labeler.SetLabel(rich)
	}), nil
}
