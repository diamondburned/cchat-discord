package discord

import (
	"context"
	"fmt"
	"strconv"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/segments"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/ningen/states/member"
)

func seekPrevGroup(l *member.List, ix int) (item, group gateway.GuildMemberListOpItem) {
	l.ViewItems(func(items []gateway.GuildMemberListOpItem) {
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

		for _, ev := range u.Ops {
			switch ev.Op {
			case "SYNC":
				ch.checkSync(c)

			case "INSERT", "UPDATE":
				item, group := seekPrevGroup(l, ev.Index)
				if item.Member != nil && group.Group != nil {
					c.SetMember(group.Group.ID, NewListMember(ev.Index, ch, item))
				}

			case "DELETE":
				_, group := seekPrevGroup(l, ev.Index-1)
				if group.Group != nil {
					c.RemoveMember(group.Group.ID, strconv.Itoa(ev.Index))
				}
			}
		}
	})

	ch.session.MemberState.RequestMemberList(ch.guildID, ch.id, 0)
	return cancel, nil
}

func (ch *Channel) checkSync(c cchat.MemberListContainer) {
	l, err := ch.session.MemberState.GetMemberList(ch.guildID, ch.id)
	if err != nil {
		ch.session.MemberState.RequestMemberList(ch.guildID, ch.id, 0)
		return
	}

	var sectionKeys []string
	var sectionsMap map[string]*ListSection

	l.ViewGroups(func(groups []gateway.GuildMemberListGroup) {
		sectionKeys = make([]string, 0, len(groups))
		sectionsMap = make(map[string]*ListSection, len(groups))

		for _, group := range groups {
			sectionKeys = append(sectionKeys, group.ID)
			sectionsMap[group.ID] = NewListSection(l.ID(), ch, group)
		}

		var sections = make([]cchat.MemberListSection, len(sectionKeys))
		for i, key := range sectionKeys {
			sections[i] = sectionsMap[key]
		}

		c.SetSections(sections)
	})

	l.ViewItems(func(items []gateway.GuildMemberListOpItem) {
		var group gateway.GuildMemberListGroup

		for i, item := range items {
			switch {
			case item.Group != nil:
				group = *item.Group

			case item.Member != nil:
				c.SetMember(group.ID, NewListMember(i, ch, item))
			}
		}
	})
}

type ListMember struct {
	ix int // experiment with this being the index

	// Keep stateful references to do on-demand loading.
	channel *Channel

	// constant states
	userID   discord.UserID
	roleID   discord.RoleID
	origName string // use if cache is stale
}

var _ cchat.ListMember = (*ListMember)(nil)

// NewListMember creates a new list member. it.Member must not be nil.
func NewListMember(ix int, ch *Channel, it gateway.GuildMemberListOpItem) *ListMember {
	roleID, _ := discord.ParseSnowflake(it.Member.HoistedRole)

	return &ListMember{
		ix:      ix,
		channel: ch,
		userID:  it.Member.User.ID,
		roleID:  discord.RoleID(roleID),
	}
}

func (l *ListMember) ID() string {
	return strconv.Itoa(l.ix)
}

func (l *ListMember) Name() text.Rich {
	n, err := l.channel.session.MemberDisplayName(l.channel.guildID, l.userID)
	if err != nil {
		return text.Plain(l.origName)
	}

	r, err := l.channel.session.State.Role(l.channel.guildID, l.roleID)
	if err != nil {
		return text.Plain(l.origName)
	}

	return text.Rich{
		Content:  n,
		Segments: []text.Segment{segments.NewColored(len(n), uint32(r.Color))},
	}
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
		return segments.FormatActivity(*p.Game)
	}

	if len(p.Activities) > 0 {
		return segments.FormatActivity(p.Activities[0])
	}

	return text.Plain("")
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

func (s *ListSection) ID() string {
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
