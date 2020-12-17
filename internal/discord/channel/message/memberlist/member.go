package memberlist

import (
	"context"
	"fmt"
	"strings"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/channel/shared"
	"github.com/diamondburned/cchat-discord/internal/segments/colored"
	"github.com/diamondburned/cchat-discord/internal/segments/emoji"
	"github.com/diamondburned/cchat-discord/internal/segments/mention"
	"github.com/diamondburned/cchat-discord/internal/urlutils"
	"github.com/diamondburned/cchat/text"
)

type Member struct {
	channel  shared.Channel
	userID   discord.UserID
	origName string // use if cache is stale
}

// New creates a new list member. it.Member must not be nil.
func NewMember(ch shared.Channel, opItem gateway.GuildMemberListOpItem) cchat.ListMember {
	return &Member{
		channel:  ch,
		userID:   opItem.Member.User.ID,
		origName: opItem.Member.User.Username,
	}
}

func (l *Member) ID() cchat.ID {
	return l.userID.String()
}

func (l *Member) Name() text.Rich {
	g, err := l.channel.State.Store.Guild(l.channel.GuildID)
	if err != nil {
		return text.Plain(l.origName)
	}

	m, err := l.channel.State.Store.Member(l.channel.GuildID, l.userID)
	if err != nil {
		return text.Plain(l.origName)
	}

	var name = m.User.Username
	if m.Nick != "" {
		name = m.Nick
	}

	mention := mention.MemberSegment(0, len(name), *g, *m)
	mention.WithState(l.channel.State.State)

	var txt = text.Rich{
		Content:  name,
		Segments: []text.Segment{mention},
	}

	if c := discord.MemberColor(*g, *m); c != discord.DefaultMemberColor {
		txt.Segments = append(txt.Segments, colored.New(len(name), uint32(c)))
	}

	return txt
}

func (l *Member) AsIconer() cchat.Iconer { return l }

func (l *Member) Icon(ctx context.Context, c cchat.IconContainer) (func(), error) {
	m, err := l.channel.State.Member(l.channel.GuildID, l.userID)
	if err != nil {
		return nil, err
	}

	c.SetIcon(urlutils.AvatarURL(m.User.AvatarURL()))

	return func() {}, nil
}

func (l *Member) Status() cchat.Status {
	p, err := l.channel.State.Store.Presence(l.channel.GuildID, l.userID)
	if err != nil {
		return cchat.StatusUnknown
	}

	switch p.Status {
	case discord.OnlineStatus:
		return cchat.StatusOnline
	case discord.DoNotDisturbStatus:
		return cchat.StatusBusy
	case discord.IdleStatus:
		return cchat.StatusAway
	case discord.OfflineStatus, discord.InvisibleStatus:
		return cchat.StatusOffline
	default:
		return cchat.StatusUnknown
	}
}

func (l *Member) Secondary() text.Rich {
	p, err := l.channel.State.Store.Presence(l.channel.GuildID, l.userID)
	if err != nil {
		return text.Plain("")
	}

	if p.Game != nil {
		return formatSmallActivity(*p.Game)
	}

	if len(p.Activities) > 0 {
		return formatSmallActivity(p.Activities[0])
	}

	return text.Plain("")
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
