package segments

import (
	"github.com/diamondburned/cchat/text"
	"github.com/yuin/goldmark/ast"
)

type BlockquoteSegment struct {
	start, end int
}

var _ text.Quoteblocker = (*BlockquoteSegment)(nil)

func (r *TextRenderer) blockquote(n *ast.Blockquote, enter bool) ast.WalkStatus {
	if enter {
		r.startBlock()
		defer r.endBlock()

		// Create a segment.
		var seg = BlockquoteSegment{start: r.i()}

		// A blockquote contains a paragraph each line. Because Discord.
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			r.buf.WriteString("> ")

			ast.Walk(child, func(node ast.Node, enter bool) (ast.WalkStatus, error) {
				// We only call when entering, since we don't want to trigger a
				// hard new line after each paragraph.
				if enter {
					return r.renderNode(node, enter)
				}
				return ast.WalkContinue, nil
			})
		}

		// Write the end of the segment.
		seg.end = r.i()
		r.append(seg)
	}

	return ast.WalkSkipChildren
}

func (b BlockquoteSegment) Bounds() (start, end int) {
	return b.start, b.end
}

func (b BlockquoteSegment) Quote() {}
