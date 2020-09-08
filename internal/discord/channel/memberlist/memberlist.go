package memberlist

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/state"
	"github.com/diamondburned/ningen/states/member"
)

type Channel struct {
	// Keep stateful references to do on-demand loading.
	state *state.Instance
	// constant states
	channelID discord.ChannelID
	guildID   discord.GuildID
}

func NewChannel(s *state.Instance, ch discord.ChannelID, g discord.GuildID) Channel {
	return Channel{
		state:     s,
		channelID: ch,
		guildID:   g,
	}
}

func (ch Channel) FlushMemberGroups(l *member.List, c cchat.MemberListContainer) {
	l.ViewGroups(func(groups []gateway.GuildMemberListGroup) {
		var sections = make([]cchat.MemberSection, len(groups))
		for i, group := range groups {
			sections[i] = ch.NewSection(l.ID(), group)
		}

		c.SetSections(sections)
	})
}
