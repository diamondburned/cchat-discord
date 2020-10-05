package blockquote

import (
	"github.com/diamondburned/cchat-discord/internal/segments/renderer"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/cchat/utils/empty"
	"github.com/yuin/goldmark/ast"
)

func init() {
	renderer.Register(ast.KindBlockquote, blockquote)
}

func blockquote(r *renderer.Text, node ast.Node, enter bool) ast.WalkStatus {
	n := node.(*ast.Blockquote)

	if enter {
		// Block formatting.
		r.EnsureBreak()
		defer r.EnsureBreak()

		// Create a segment.
		var seg = Segment{Start: r.Buffer.Len()}

		// A blockquote contains a paragraph each line. Because Discord.
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			r.Buffer.WriteString("> ")

			ast.Walk(child, func(node ast.Node, enter bool) (ast.WalkStatus, error) {
				// We only call when entering, since we don't want to trigger a
				// hard new line after each paragraph.
				if enter {
					return r.RenderNode(node, enter)
				}
				return ast.WalkContinue, nil
			})
		}

		// Search until the last non-whitespace.
		var i = r.Buffer.Len() - 1
		for bytes := r.Buffer.Bytes(); i > 0 && isSpace(bytes[i]); i-- {
		}

		// The ending will have a trailing character that's not covered, so
		// we'll need to do that ourselves.
		// End the codeblock at that non-whitespace location.
		seg.End = i + 1
		r.Append(seg)
	}

	return ast.WalkSkipChildren
}

type Segment struct {
	empty.TextSegment
	Start, End int
}

var (
	_ text.Segment      = (*Segment)(nil)
	_ text.Quoteblocker = (*Segment)(nil)
)

func (b Segment) Bounds() (start, end int) {
	return b.Start, b.End
}

func (b Segment) AsBlockquoter() text.Quoteblocker {
	return b
}

func (b Segment) QuotePrefix() string {
	return "> "
}

// isSpace is a quick function that matches if the byte is a space, a new line
// or a return carriage.
func isSpace(b byte) bool {
	return b == ' ' || b == '\n' || b == '\r'
}
