package memberlist

import (
	"context"

	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/channel/shared"
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

type MemberLister struct {
	*shared.Channel
}

func New(ch *shared.Channel) cchat.MemberLister {
	return MemberLister{ch}
}

func (ml MemberLister) ListMembers(ctx context.Context, c cchat.MemberListContainer) (func(), error) {
	if !ml.GuildID.IsValid() {
		return func() {}, nil
	}

	cancel := ml.State.AddHandler(func(u *gateway.GuildMemberListUpdate) {
		l, err := ml.State.MemberState.GetMemberList(ml.GuildID, ml.ID)
		if err != nil {
			return // wat
		}

		if l.GuildID() != u.GuildID || l.ID() != u.ID {
			return
		}

		for _, ev := range u.Ops {
			switch ev.Op {
			case "SYNC":
				ml.checkSync(c)

			case "INSERT", "UPDATE":
				item, group := seekPrevGroup(l, ev.Index)
				if item.Member != nil && group.Group != nil {
					c.SetMember(group.Group.ID, NewMember(ml.Channel, item))
					ml.FlushMemberGroups(l, c)
				}

			case "DELETE":
				_, group := seekPrevGroup(l, ev.Index-1)
				if group.Group != nil && ev.Item.Member != nil {
					c.RemoveMember(group.Group.ID, ev.Item.Member.User.ID.String())
					ml.FlushMemberGroups(l, c)
				}
			}
		}
	})

	ml.checkSync(c)

	return cancel, nil
}

func (ml MemberLister) checkSync(c cchat.MemberListContainer) {
	l, err := ml.State.MemberState.GetMemberList(ml.GuildID, ml.ID)
	if err != nil {
		ml.State.MemberState.RequestMemberList(ml.GuildID, ml.ID, 0)
		return
	}

	ml.FlushMemberGroups(l, c)

	l.ViewItems(func(items []gateway.GuildMemberListOpItem) {
		var group gateway.GuildMemberListGroup

		for _, item := range items {
			switch {
			case item.Group != nil:
				group = *item.Group

			case item.Member != nil:
				c.SetMember(group.ID, NewMember(ml.Channel, item))
			}
		}
	})
}

func (ml MemberLister) FlushMemberGroups(l *member.List, c cchat.MemberListContainer) {
	l.ViewGroups(func(groups []gateway.GuildMemberListGroup) {
		var sections = make([]cchat.MemberSection, len(groups))
		for i, group := range groups {
			sections[i] = NewSection(ml.Channel, l.ID(), group)
		}

		c.SetSections(sections)
	})
}
