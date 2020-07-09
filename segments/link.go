package segments

import (
	"github.com/diamondburned/cchat/text"
	"github.com/yuin/goldmark/ast"
)

// linkState is used for ast.Link segments.
type linkState struct {
	linkstack []int // stack of starting integers
}

type LinkSegment struct {
	start, end int
	url        string
}

var _ text.Linker = (*LinkSegment)(nil)

func (r *TextRenderer) link(n *ast.Link, enter bool) ast.WalkStatus {
	// If we're entering the link node, then add the starting point to the stack
	// and move on.
	if enter {
		r.lnks.linkstack = append(r.lnks.linkstack, r.buf.Len())
		return ast.WalkContinue
	}

	// If there's nothing in the stack, then don't do anything. This shouldn't
	// happen.
	if len(r.lnks.linkstack) == 0 {
		return ast.WalkContinue
	}

	// We're exiting the link node. Pop the segment off the stack.
	ilast := len(r.lnks.linkstack) - 1
	start := r.lnks.linkstack[ilast]
	r.lnks.linkstack = r.lnks.linkstack[:ilast]

	// Close the segment on enter false.
	r.append(LinkSegment{
		start,
		r.buf.Len(),
		string(n.Destination),
	})

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
