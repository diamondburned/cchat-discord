package memberlist

import (
	"context"
	"fmt"
	"strings"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/channel/shared"
	"github.com/diamondburned/cchat-discord/internal/segments/emoji"
	"github.com/diamondburned/cchat-discord/internal/segments/mention"
	"github.com/diamondburned/cchat/text"
)

type Member struct {
	channel  shared.Channel
	mention  mention.User
	presence gateway.Presence
}

// New creates a new list member. it.Member must not be nil.
func NewMember(ch shared.Channel, opItem gateway.GuildMemberListOpItem) cchat.ListMember {
	user := mention.NewUser(opItem.Member.User)
	user.WithState(ch.State.State)
	user.WithMember(opItem.Member.Member)
	user.WithGuildID(ch.GuildID)
	user.WithPresence(opItem.Member.Presence)
	user.Prefetch()

	return &Member{
		channel:  ch,
		presence: opItem.Member.Presence,
		mention:  *user,
	}
}

func (l *Member) ID() cchat.ID {
	return l.mention.UserID().String()
}

func (l *Member) Name() text.Rich {
	content := l.mention.DisplayName()

	return text.Rich{
		Content: content,
		Segments: []text.Segment{
			mention.NewSegment(0, len(content), &l.mention),
		},
	}
}

func (l *Member) AsIconer() cchat.Iconer { return l }

func (l *Member) Icon(ctx context.Context, c cchat.IconContainer) (func(), error) {
	c.SetIcon(l.mention.Avatar())
	return func() {}, nil
}

func (l *Member) Status() cchat.Status {
	switch l.presence.Status {
	case gateway.OnlineStatus:
		return cchat.StatusOnline
	case gateway.DoNotDisturbStatus:
		return cchat.StatusBusy
	case gateway.IdleStatus:
		return cchat.StatusAway
	case gateway.OfflineStatus, gateway.InvisibleStatus:
		return cchat.StatusOffline
	default:
		return cchat.StatusUnknown
	}
}

func (l *Member) Secondary() text.Rich {
	if len(l.presence.Activities) == 0 {
		return text.Rich{}
	}

	return formatSmallActivity(l.presence.Activities[0])
}

func formatSmallActivity(ac discord.Activity) text.Rich {
	switch ac.Type {
	case discord.GameActivity:
		return text.Plain(fmt.Sprintf("Playing %s", ac.Name))

	case discord.ListeningActivity:
		return text.Plain(fmt.Sprintf("Listening to %s", ac.Name))

	case discord.StreamingActivity:
		return text.Plain(fmt.Sprintf("Streaming on %s", ac.Name))

	case discord.CustomActivity:
		var status strings.Builder
		var segmts []text.Segment

		if ac.Emoji != nil {
			if !ac.Emoji.ID.IsValid() {
				status.WriteString(ac.Emoji.Name)
				status.WriteByte(' ')
			} else {
				segmts = append(segmts, emoji.Segment{
					Start: status.Len(),
					Emoji: emoji.EmojiFromDiscord(*ac.Emoji, ac.State == ""),
				})
			}
		}

		status.WriteString(ac.State)

		return text.Rich{
			Content:  status.String(),
			Segments: segmts,
		}

	default:
		return text.Rich{}
	}
}
