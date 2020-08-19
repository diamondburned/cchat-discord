package discord

import (
	"context"
	"fmt"
	"strings"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/segments"
	"github.com/diamondburned/cchat-discord/urlutils"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/ningen/states/member"
)

func seekPrevGroup(l *member.List, ix int) (item, group gateway.GuildMemberListOpItem) {
	l.ViewItems(func(items []gateway.GuildMemberListOpItem) {
		// Bound check.
		if ix >= len(items) {
			return
		}

		item = items[ix]

		// Search backwards.
		for i := ix; i >= 0; i-- {
			if items[i].Group != nil {
				group = items[i]
				return
			}
		}
	})

	return
}

func (ch *Channel) ListMembers(ctx context.Context, c cchat.MemberListContainer) (func(), error) {
	if !ch.guildID.IsValid() {
		return func() {}, nil
	}

	cancel := ch.session.AddHandler(func(u *gateway.GuildMemberListUpdate) {
		l, err := ch.session.MemberState.GetMemberList(ch.guildID, ch.id)
		if err != nil {
			return // wat
		}

		if l.GuildID() != u.GuildID || l.ID() != u.ID {
			return
		}

		for _, ev := range u.Ops {
			switch ev.Op {
			case "SYNC":
				ch.checkSync(c)

			case "INSERT", "UPDATE":
				item, group := seekPrevGroup(l, ev.Index)
				if item.Member != nil && group.Group != nil {
					c.SetMember(group.Group.ID, NewListMember(ch, item))
					ch.flushMemberGroups(l, c)
				}

			case "DELETE":
				_, group := seekPrevGroup(l, ev.Index-1)
				if group.Group != nil && ev.Item.Member != nil {
					c.RemoveMember(group.Group.ID, ev.Item.Member.User.ID.String())
					ch.flushMemberGroups(l, c)
				}
			}
		}
	})

	ch.checkSync(c)

	return cancel, nil
}

func (ch *Channel) checkSync(c cchat.MemberListContainer) {
	l, err := ch.session.MemberState.GetMemberList(ch.guildID, ch.id)
	if err != nil {
		ch.session.MemberState.RequestMemberList(ch.guildID, ch.id, 0)
		return
	}

	ch.flushMemberGroups(l, c)

	l.ViewItems(func(items []gateway.GuildMemberListOpItem) {
		var group gateway.GuildMemberListGroup

		for _, item := range items {
			switch {
			case item.Group != nil:
				group = *item.Group

			case item.Member != nil:
				c.SetMember(group.ID, NewListMember(ch, item))
			}
		}
	})
}

func (ch *Channel) flushMemberGroups(l *member.List, c cchat.MemberListContainer) {
	l.ViewGroups(func(groups []gateway.GuildMemberListGroup) {
		var sections = make([]cchat.MemberListSection, len(groups))
		for i, group := range groups {
			sections[i] = NewListSection(l.ID(), ch, group)
		}

		c.SetSections(sections)
	})
}

type ListMember struct {
	// Keep stateful references to do on-demand loading.
	channel *Channel

	// constant states
	userID   discord.UserID
	origName string // use if cache is stale
}

var (
	_ cchat.ListMember = (*ListMember)(nil)
	_ cchat.Icon       = (*ListMember)(nil)
)

// NewListMember creates a new list member. it.Member must not be nil.
func NewListMember(ch *Channel, it gateway.GuildMemberListOpItem) *ListMember {
	return &ListMember{
		channel:  ch,
		userID:   it.Member.User.ID,
		origName: it.Member.User.Username,
	}
}

func (l *ListMember) ID() cchat.ID {
	return l.userID.String()
}

func (l *ListMember) Name() text.Rich {
	g, err := l.channel.guild()
	if err != nil {
		return text.Plain(l.origName)
	}

	m, err := l.channel.session.Member(l.channel.guildID, l.userID)
	if err != nil {
		return text.Plain(l.origName)
	}

	var name = m.User.Username
	if m.Nick != "" {
		name = m.Nick
	}

	mention := segments.MemberSegment(0, len(name), *g, *m)
	mention.WithState(l.channel.session.State)

	var txt = text.Rich{
		Content:  name,
		Segments: []text.Segment{mention},
	}

	if c := discord.MemberColor(*g, *m); c != discord.DefaultMemberColor {
		txt.Segments = append(txt.Segments, segments.NewColored(len(name), uint32(c)))
	}

	return txt
}

func (l *ListMember) Icon(ctx context.Context, c cchat.IconContainer) (func(), error) {
	m, err := l.channel.session.Member(l.channel.guildID, l.userID)
	if err != nil {
		return nil, err
	}

	c.SetIcon(urlutils.AvatarURL(m.User.AvatarURL()))

	return func() {}, nil
}

func (l *ListMember) Status() cchat.UserStatus {
	p, err := l.channel.session.State.Presence(l.channel.guildID, l.userID)
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

func (l *ListMember) Secondary() text.Rich {
	p, err := l.channel.session.State.Presence(l.channel.guildID, l.userID)
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

type ListSection struct {
	// constant states
	listID string
	id     string // roleID or online or offline
	name   string
	total  int

	channel *Channel
}

var (
	_ cchat.MemberListSection        = (*ListSection)(nil)
	_ cchat.MemberListDynamicSection = (*ListSection)(nil)
)

func NewListSection(listID string, ch *Channel, group gateway.GuildMemberListGroup) *ListSection {
	var name string

	switch group.ID {
	case "online":
		name = "Online"
	case "offline":
		name = "Offline"
	default:
		p, err := discord.ParseSnowflake(group.ID)
		if err != nil {
			name = group.ID
		} else {
			r, err := ch.session.Role(ch.guildID, discord.RoleID(p))
			if err != nil {
				name = fmt.Sprintf("<@#%s>", p.String())
			} else {
				name = r.Name
			}
		}
	}

	return &ListSection{
		listID:  listID,
		channel: ch,
		id:      group.ID,
		name:    name,
		total:   int(group.Count),
	}
}

func (s *ListSection) ID() cchat.ID {
	return s.id
	// return fmt.Sprintf("%s-%s", s.listID, s.name)
}

func (s *ListSection) Name() text.Rich {
	return text.Rich{Content: s.name}
}

func (s *ListSection) Total() int {
	return s.total
}

// TODO: document that Load{More,Less} works more like a shifting window.

func (s *ListSection) LoadMore() bool {
	// This variable is here purely to make lines shorter.
	var memstate = s.channel.session.MemberState

	chunk := memstate.GetMemberListChunk(s.channel.guildID, s.channel.id)
	if chunk < 0 {
		chunk = 0
	}

	return memstate.RequestMemberList(s.channel.guildID, s.channel.id, chunk) != nil
}

func (s *ListSection) LoadLess() bool {
	var memstate = s.channel.session.MemberState

	chunk := memstate.GetMemberListChunk(s.channel.guildID, s.channel.id)
	if chunk <= 0 {
		return false
	}

	memstate.RequestMemberList(s.channel.guildID, s.channel.id, chunk-1)
	return true
}
