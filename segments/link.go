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
		// Write the actual title.
		start, end := r.write(n.Title)

		// Close the segment.
		r.append(LinkSegment{
			start,
			end,
			string(n.Destination),
		})
	}

	return ast.WalkContinue
}

func (r *TextRenderer) autoLink(n *ast.AutoLink, enter bool) ast.WalkStatus {
	if enter {
		start, end := r.write(n.URL(r.src))

		r.append(LinkSegment{
			start,
			end,
			string(n.URL((r.src))),
		})
	}

	return ast.WalkContinue
}

func (l LinkSegment) Bounds() (start, end int) {
	return l.start, l.end
}

func (l LinkSegment) Link() (url string) {
	return l.url
}
