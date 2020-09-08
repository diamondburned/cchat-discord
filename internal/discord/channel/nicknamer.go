package channel

import (
	"context"

	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/segments"
	"github.com/diamondburned/cchat/text"
	"github.com/pkg/errors"
)

var _ cchat.Nicknamer = (*Channel)(nil)

// IsNicknamer returns true if the current channel is in a guild.
func (ch *Channel) IsNicknamer() bool {
	return ch.guildID.IsValid()
}

func (ch *Channel) Nickname(ctx context.Context, labeler cchat.LabelContainer) (func(), error) {
	// We don't have a nickname if we're not in a guild.
	if !ch.guildID.IsValid() {
		return func() {}, nil
	}

	state := ch.state.WithContext(ctx)

	// MemberColor should fill up the state cache.
	c, err := state.MemberColor(ch.guildID, ch.state.UserID)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get self member color")
	}

	m, err := state.Member(ch.guildID, ch.state.UserID)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get self member")
	}

	var rich = text.Rich{Content: m.User.Username}
	if m.Nick != "" {
		rich.Content = m.Nick
	}
	if c > 0 {
		rich.Segments = []text.Segment{
			segments.NewColored(len(rich.Content), c.Uint32()),
		}
	}

	labeler.SetLabel(rich)

	// Copy the user ID to use.
	var selfID = m.User.ID

	return ch.state.AddHandler(func(g *gateway.GuildMemberUpdateEvent) {
		if g.GuildID != ch.guildID || g.User.ID != selfID {
			return
		}

		var rich = text.Rich{Content: m.User.Username}
		if m.Nick != "" {
			rich.Content = m.Nick
		}

		c, err := ch.state.MemberColor(g.GuildID, selfID)
		if err == nil {
			rich.Segments = []text.Segment{
				segments.NewColored(len(rich.Content), c.Uint32()),
			}
		}

		labeler.SetLabel(rich)
	}), nil
}
