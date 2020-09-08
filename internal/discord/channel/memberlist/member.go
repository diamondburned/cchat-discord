package memberlist

import (
	"context"
	"fmt"
	"strings"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/state"
	"github.com/diamondburned/cchat-discord/internal/segments"
	"github.com/diamondburned/cchat-discord/internal/urlutils"
	"github.com/diamondburned/cchat/text"
)

type Member struct {
	Channel
	state *state.Instance

	userID   discord.UserID
	origName string // use if cache is stale
}

var (
	_ cchat.ListMember = (*Member)(nil)
	_ cchat.Iconer     = (*Member)(nil)
)

// New creates a new list member. it.Member must not be nil.
func (c Channel) NewMember(opItem gateway.GuildMemberListOpItem) *Member {
	return &Member{
		Channel:  c,
		userID:   opItem.Member.User.ID,
		origName: opItem.Member.User.Username,
	}
}

func (l *Member) ID() cchat.ID {
	return l.userID.String()
}

func (l *Member) Name() text.Rich {
	g, err := l.state.Store.Guild(l.guildID)
	if err != nil {
		return text.Plain(l.origName)
	}

	m, err := l.state.Store.Member(l.guildID, l.userID)
	if err != nil {
		return text.Plain(l.origName)
	}

	var name = m.User.Username
	if m.Nick != "" {
		name = m.Nick
	}

	mention := segments.MemberSegment(0, len(name), *g, *m)
	mention.WithState(l.state.State)

	var txt = text.Rich{
		Content:  name,
		Segments: []text.Segment{mention},
	}

	if c := discord.MemberColor(*g, *m); c != discord.DefaultMemberColor {
		txt.Segments = append(txt.Segments, segments.NewColored(len(name), uint32(c)))
	}

	return txt
}

// IsIconer returns true.
func (l *Member) IsIconer() bool { return true }

func (l *Member) Icon(ctx context.Context, c cchat.IconContainer) (func(), error) {
	m, err := l.state.Member(l.guildID, l.userID)
	if err != nil {
		return nil, err
	}

	c.SetIcon(urlutils.AvatarURL(m.User.AvatarURL()))

	return func() {}, nil
}

func (l *Member) Status() cchat.UserStatus {
	p, err := l.state.Store.Presence(l.guildID, l.userID)
	if err != nil {
		return cchat.UnknownStatus
	}

	switch p.Status {
	case discord.OnlineStatus:
		return cchat.OnlineStatus
	case discord.DoNotDisturbStatus:
		return cchat.BusyStatus
	case discord.IdleStatus:
		return cchat.AwayStatus
	case discord.OfflineStatus, discord.InvisibleStatus:
		return cchat.OfflineStatus
	default:
		return cchat.UnknownStatus
	}
}

func (l *Member) Secondary() text.Rich {
	p, err := l.state.Store.Presence(l.guildID, l.userID)
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
				segmts = append(segmts, segments.EmojiSegment{
					Start:    status.Len(),
					Name:     ac.Emoji.Name,
					EmojiURL: ac.Emoji.EmojiURL() + "?size=64",
					Large:    ac.State == "",
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
