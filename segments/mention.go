package segments

import (
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/ningen/md"
	"github.com/yuin/goldmark/ast"
)

const (
	mentionChannel uint8 = iota
	mentionUser
	mentionRole
)

type MentionSegment struct {
	start, end int
}

var _ text.Segment = (*MentionSegment)(nil)

func (r *TextRenderer) mention(n *md.Mention, enter bool) ast.WalkStatus {
	if enter {
		seg := MentionSegment{start: r.i()}

		switch {
		case n.Channel != nil:
			r.buf.WriteString("#" + n.Channel.Name)
		case n.GuildUser != nil:
			r.buf.WriteString("@" + n.GuildUser.Username)
		case n.GuildRole != nil:
			r.buf.WriteString("@" + n.GuildRole.Name)
		}

		seg.end = r.i()
		r.append(seg)
	}

	return ast.WalkContinue
}

func (m MentionSegment) Bounds() (start, end int) {
	return m.start, m.end
}

// TODO
func (m MentionSegment) MentionInfo() text.Rich {
	return text.Rich{}
}
