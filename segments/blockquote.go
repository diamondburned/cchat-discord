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
		// Block formatting.
		r.ensureBreak()
		defer r.ensureBreak()

		// Create a segment.
		var seg = BlockquoteSegment{start: r.buf.Len()}

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

		// Search until the last non-whitespace.
		var i = r.buf.Len() - 1
		for bytes := r.buf.Bytes(); i > 0 && isSpace(bytes[i]); i-- {
		}

		// The ending will have a trailing character that's not covered, so
		// we'll need to do that ourselves.
		// End the codeblock at that non-whitespace location.
		seg.end = i + 1
		r.append(seg)
	}

	return ast.WalkSkipChildren
}

func (b BlockquoteSegment) Bounds() (start, end int) {
	return b.start, b.end
}

func (b BlockquoteSegment) Quote() {}

// isSpace is a quick function that matches if the byte is a space, a new line
// or a return carriage.
func isSpace(b byte) bool {
	return b == ' ' || b == '\n' || b == '\r'
}
