package mention

import (
	"github.com/diamondburned/cchat-discord/internal/segments/renderer"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/cchat/utils/empty"
	"github.com/diamondburned/ningen/md"
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
			seg.User = NewUser(r.Store, r.Message.GuildID, *n.GuildUser)
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
	case s.User != nil:
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
