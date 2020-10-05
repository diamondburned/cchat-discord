package link

import (
	"github.com/diamondburned/cchat-discord/internal/segments/renderer"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/cchat/utils/empty"
	"github.com/yuin/goldmark/ast"
)

func init() {
	renderer.Register(ast.KindLink, link)
	renderer.Register(ast.KindAutoLink, autoLink)
}

func link(r *renderer.Text, node ast.Node, enter bool) ast.WalkStatus {
	n := node.(*ast.Link)

	// If we're entering the link node, then add the starting point to the stack
	// and move on.
	if enter {
		r.Links.Append(r.Buffer.Len())
		return ast.WalkContinue
	}

	// If there's nothing in the stack, then don't do anything. This shouldn't
	// happen.
	if r.Links.Len() == 0 {
		return ast.WalkContinue
	}

	// We're exiting the link node. Pop the segment off the stack.
	start := r.Links.Pop()

	// Close the segment on enter false.
	r.Append(NewSegment(start, r.Buffer.Len(), string(n.Destination)))

	return ast.WalkContinue
}

func autoLink(r *renderer.Text, node ast.Node, enter bool) ast.WalkStatus {
	n := node.(*ast.AutoLink)

	if enter {
		start, end := r.Write(n.URL(r.Source))
		r.Append(NewSegment(start, end, string(n.URL(r.Source))))
	}

	return ast.WalkContinue
}

type URL string

var _ text.Linker = (*URL)(nil)

func (u URL) Link() string { return string(u) }

type Segment struct {
	empty.TextSegment
	start int
	end   int
	url   URL
}

var _ text.Segment = (*Segment)(nil)

func NewSegment(start, end int, url string) Segment {
	return Segment{
		start: start,
		end:   end,
		url:   URL(url),
	}
}

func (l Segment) Bounds() (start, end int) {
	return l.start, l.end
}

func (l Segment) AsLinker() text.Linker {
	return l.url
}
