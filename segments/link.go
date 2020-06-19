package segments

import (
	"github.com/diamondburned/cchat/text"
	"github.com/yuin/goldmark/ast"
)

type LinkSegment struct {
	start, end int
	url        string
}

var _ text.Linker = (*LinkSegment)(nil)

func (r *TextRenderer) link(n *ast.Link, enter bool) ast.WalkStatus {
	if enter {
		// Make a segment with a start pointing to the end of buffer.
		seg := LinkSegment{
			start: r.i(),
			url:   string(n.Destination),
		}

		// Write the actual title.
		r.buf.Write(n.Title)

		// Close the segment.
		seg.end = r.i()
		r.append(seg)
	}

	return ast.WalkContinue
}

func (r *TextRenderer) autoLink(n *ast.AutoLink, enter bool) ast.WalkStatus {
	if enter {
		seg := LinkSegment{
			start: r.i(),
			url:   string(n.URL(r.src)),
		}

		r.buf.Write(n.URL(r.src))

		seg.end = r.i()
		r.append(seg)
	}

	return ast.WalkContinue
}

func (l LinkSegment) Bounds() (start, end int) {
	return l.start, l.end
}

func (l LinkSegment) Link() (url string) {
	return l.url
}
