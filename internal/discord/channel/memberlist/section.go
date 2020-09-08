package memberlist

import (
	"fmt"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat/text"
)

type Section struct {
	Channel

	// constant states
	listID string
	id     string // roleID or online or offline
	name   string
	total  int
}

var (
	_ cchat.MemberSection        = (*Section)(nil)
	_ cchat.MemberDynamicSection = (*Section)(nil)
)

func (ch Channel) NewSection(listID string, group gateway.GuildMemberListGroup) *Section {
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
			r, err := ch.state.Role(ch.guildID, discord.RoleID(p))
			if err != nil {
				name = fmt.Sprintf("<@#%s>", p.String())
			} else {
				name = r.Name
			}
		}
	}

	return &Section{
		Channel: ch,
		listID:  listID,
		id:      group.ID,
		name:    name,
		total:   int(group.Count),
	}
}

func (s *Section) ID() cchat.ID {
	return s.id
}

func (s *Section) Name() text.Rich {
	return text.Rich{Content: s.name}
}

func (s *Section) Total() int {
	return s.total
}

func (s *Section) IsMemberDynamicSection() bool { return true }

// TODO: document that Load{More,Less} works more like a shifting window.

func (s *Section) LoadMore() bool {
	chunk := s.state.MemberState.GetMemberListChunk(s.guildID, s.channelID)
	if chunk < 0 {
		chunk = 0
	}

	return s.state.MemberState.RequestMemberList(s.guildID, s.channelID, chunk) != nil
}

func (s *Section) LoadLess() bool {
	chunk := s.state.MemberState.GetMemberListChunk(s.guildID, s.channelID)
	if chunk <= 0 {
		return false
	}

	s.state.MemberState.RequestMemberList(s.guildID, s.channelID, chunk-1)
	return true
}
