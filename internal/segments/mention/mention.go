package mention

import (
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/cchat-discord/internal/segments/renderer"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/cchat/utils/empty"
	"github.com/diamondburned/ningen/v2/md"
	"github.com/yuin/goldmark/ast"
)

func init() {
	renderer.Register(md.KindMention, mention)
}

func mention(r *renderer.Text, node ast.Node, enter bool) ast.WalkStatus {
	n := node.(*md.Mention)

	if enter {
		var seg = Segment{}

		switch {
		case n.Channel != nil:
			seg.Start, seg.End = r.WriteString("#" + n.Channel.Name)
			seg.Channel = NewChannel(*n.Channel)

		case n.GuildUser != nil:
			seg.Start, seg.End = r.WriteString("@" + n.GuildUser.Username)
			seg.User = NewUser(n.GuildUser.User)
			seg.User.store = r.Store
			seg.User.WithGuildID(r.Message.GuildID)
			if n.GuildUser.Member != nil {
				seg.User.WithMember(*n.GuildUser.Member)
			}
			seg.User.Prefetch()

		case n.GuildRole != nil:
			seg.Start, seg.End = r.WriteString("@" + n.GuildRole.Name)
			seg.Role = NewRole(*n.GuildRole)

		default:
			// Unexpected error; skip.
			return ast.WalkSkipChildren
		}

		r.Append(seg)
	}

	return ast.WalkContinue
}

type Segment struct {
	empty.TextSegment
	Start, End int

	// enums?
	Channel *Channel
	User    *User
	Role    *Role
}

func (s Segment) Bounds() (start, end int) {
	return s.Start, s.End
}

func (s Segment) AsColorer() text.Colorer {
	switch {
	case s.User != nil && s.User.HasColor():
		return s.User
	case s.Role != nil:
		return s.Role
	}
	return nil
}

func (s Segment) AsAvatarer() text.Avatarer {
	switch {
	case s.User != nil:
		return s.User
	}

	return nil
}

func (s Segment) AsMentioner() text.Mentioner {
	switch {
	case s.Channel != nil:
		return s.Channel
	case s.User != nil:
		return s.User
	}
	return nil
}

func MemberColor(guild discord.Guild, member discord.Member) (c uint32, ok bool) {
	var pos int

	for _, r := range guild.Roles {
		for _, mr := range member.RoleIDs {
			if mr != r.ID {
				continue
			}

			if r.Color > 0 && r.Position > pos {
				c = r.Color.Uint32()
				ok = true
				pos = r.Position
			}
		}
	}

	return
}
