package memberlist

import (
	"fmt"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/channel/shared"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/cchat/utils/empty"
)

type Section struct {
	empty.Namer

	// constant states
	listID string
	id     string // roleID or online or offline
	name   string
	total  int
	dynsec DynamicSection
}

func NewSection(
	ch shared.Channel,
	listID string,
	group gateway.GuildMemberListGroup) cchat.MemberSection {

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
			r, err := ch.State.Role(ch.GuildID, discord.RoleID(p))
			if err != nil {
				name = fmt.Sprintf("<@#%s>", p.String())
			} else {
				name = r.Name
			}
		}
	}

	return Section{
		listID: listID,
		id:     group.ID,
		name:   name,
		total:  int(group.Count),
		dynsec: DynamicSection{
			Channel: ch,
		},
	}
}

func (s Section) ID() cchat.ID {
	return s.id
}

func (s Section) Name() text.Rich {
	return text.Rich{Content: s.name}
}

func (s Section) Total() int {
	return s.total
}

func (s Section) AsMemberDynamicSection() cchat.MemberDynamicSection {
	return s.dynsec
}

func (s Section) IsMemberDynamicSection() bool { return true }

type DynamicSection struct {
	shared.Channel
}

var _ cchat.MemberDynamicSection = (*DynamicSection)(nil)

// TODO: document that Load{More,Less} works more like a shifting window.

func (s DynamicSection) LoadMore() bool {
	chunk := s.State.MemberState.GetMemberListChunk(s.GuildID, s.Channel.ID)
	if chunk < 0 {
		chunk = 0
	}

	return s.State.MemberState.RequestMemberList(s.GuildID, s.Channel.ID, chunk) != nil
}

func (s DynamicSection) LoadLess() bool {
	chunk := s.State.MemberState.GetMemberListChunk(s.GuildID, s.Channel.ID)
	if chunk <= 0 {
		return false
	}

	s.State.MemberState.RequestMemberList(s.GuildID, s.Channel.ID, chunk-1)
	return true
}
