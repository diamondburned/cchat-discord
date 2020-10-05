package codeblock

import (
	"github.com/diamondburned/cchat-discord/internal/segments/renderer"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/cchat/utils/empty"
	"github.com/yuin/goldmark/ast"
)

func init() {
	renderer.Register(ast.KindCodeBlock, codeblock)
}

func codeblock(r *renderer.Text, node ast.Node, enter bool) ast.WalkStatus {
	n := node.(*ast.FencedCodeBlock)

	if enter {
		// Open the block by adding formatting and all.
		r.StartBlock()
		r.Buffer.WriteString("---\n")

		// Create a segment.
		seg := CodeblockSegment{
			Start:    r.Buffer.Len(),
			Language: string(n.Language(r.Source)),
		}

		// Join all lines together.
		var lines = n.Lines()

		for i := 0; i < lines.Len(); i++ {
			line := lines.At(i)
			r.Buffer.Write(line.Value(r.Source))
		}

		// Close the segment.
		seg.End = r.Buffer.Len()
		r.Append(seg)

		// Close the block.
		r.Buffer.WriteString("\n---")
		r.EndBlock()
	}

	return ast.WalkContinue
}

type CodeblockSegment struct {
	empty.TextSegment
	Start, End int
	Language   string
}

var _ text.Codeblocker = (*CodeblockSegment)(nil)

func (c CodeblockSegment) Bounds() (start, end int) {
	return c.Start, c.End
}

func (c CodeblockSegment) AsCodeblocker() text.Codeblocker {
	return c
}

func (c CodeblockSegment) CodeblockLanguage() string {
	return c.Language
}
