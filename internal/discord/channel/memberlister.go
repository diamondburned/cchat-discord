package channel

import (
	"context"

	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/channel/memberlist"
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

var _ cchat.MemberLister = (*Channel)(nil)

// IsMemberLister returns true if the channel is a guild channel.
func (ch *Channel) IsMemberLister() bool {
	return ch.guildID.IsValid()
}

func (ch *Channel) memberListCh() memberlist.Channel {
	return memberlist.NewChannel(ch.state, ch.id, ch.guildID)
}

func (ch *Channel) ListMembers(ctx context.Context, c cchat.MemberListContainer) (func(), error) {
	if !ch.guildID.IsValid() {
		return func() {}, nil
	}

	cancel := ch.state.AddHandler(func(u *gateway.GuildMemberListUpdate) {
		l, err := ch.state.MemberState.GetMemberList(ch.guildID, ch.id)
		if err != nil {
			return // wat
		}

		if l.GuildID() != u.GuildID || l.ID() != u.ID {
			return
		}

		var listCh = ch.memberListCh()

		for _, ev := range u.Ops {
			switch ev.Op {
			case "SYNC":
				ch.checkSync(c)

			case "INSERT", "UPDATE":
				item, group := seekPrevGroup(l, ev.Index)
				if item.Member != nil && group.Group != nil {
					c.SetMember(group.Group.ID, listCh.NewMember(item))
					listCh.FlushMemberGroups(l, c)
				}

			case "DELETE":
				_, group := seekPrevGroup(l, ev.Index-1)
				if group.Group != nil && ev.Item.Member != nil {
					c.RemoveMember(group.Group.ID, ev.Item.Member.User.ID.String())
					listCh.FlushMemberGroups(l, c)
				}
			}
		}
	})

	ch.checkSync(c)

	return cancel, nil
}

func (ch *Channel) checkSync(c cchat.MemberListContainer) {
	l, err := ch.state.MemberState.GetMemberList(ch.guildID, ch.id)
	if err != nil {
		ch.state.MemberState.RequestMemberList(ch.guildID, ch.id, 0)
		return
	}

	listCh := ch.memberListCh()
	listCh.FlushMemberGroups(l, c)

	l.ViewItems(func(items []gateway.GuildMemberListOpItem) {
		var group gateway.GuildMemberListGroup

		for _, item := range items {
			switch {
			case item.Group != nil:
				group = *item.Group

			case item.Member != nil:
				c.SetMember(group.ID, listCh.NewMember(item))
			}
		}
	})
}
