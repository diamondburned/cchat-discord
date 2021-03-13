package indicate

import (
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/channel/shared"
	"github.com/diamondburned/ningen/v2/states/read"
	"github.com/pkg/errors"
)

type UnreadIndicator struct {
	shared.Channel
}

func NewUnread(ch shared.Channel) cchat.UnreadIndicator {
	return UnreadIndicator{ch}
}

// Muted returns if this channel is muted. This includes the channel's category
// and guild.
func (ui UnreadIndicator) Muted() bool {
	return (ui.GuildID.IsValid() && ui.State.MutedState.Guild(ui.GuildID, false)) ||
		ui.State.MutedState.Channel(ui.ID) ||
		ui.State.MutedState.Category(ui.ID)
}

func (ui UnreadIndicator) UnreadIndicate(indicator cchat.UnreadContainer) (func(), error) {
	if rs := ui.State.ReadState.FindLast(ui.ID); rs != nil {
		c, err := ui.Self()
		if err != nil {
			return nil, errors.Wrap(err, "Failed to get self channel")
		}

		if c.LastMessageID > rs.LastMessageID && !ui.Muted() {
			indicator.SetUnread(true, rs.MentionCount > 0)
		}
	}

	return ui.State.ReadState.OnUpdate(func(ev *read.UpdateEvent) {
		if ui.ID == ev.ChannelID && !ui.Muted() {
			indicator.SetUnread(ev.Unread, ev.MentionCount > 0)
		}
	}), nil
}

func (ui UnreadIndicator) MarkRead(msgID cchat.ID) {
	d, err := discord.ParseSnowflake(msgID)
	if err != nil {
		return
	}

	ui.State.ReadState.MarkRead(ui.ID, discord.MessageID(d))
}
