package segments

import (
	"github.com/diamondburned/cchat/text"
	"github.com/yuin/goldmark/ast"
)

type CodeblockSegment struct {
	start, end int
	language   string
}

var _ text.Codeblocker = (*CodeblockSegment)(nil)

func (r *TextRenderer) codeblock(n *ast.FencedCodeBlock, enter bool) ast.WalkStatus {
	if enter {
		// Open the block by adding formatting and all.
		r.startBlock()
		r.buf.WriteString("---\n")

		// Create a segment.
		seg := CodeblockSegment{
			start:    r.buf.Len(),
			language: string(n.Language(r.src)),
		}

		// Join all lines together.
		var lines = n.Lines()

		for i := 0; i < lines.Len(); i++ {
			line := lines.At(i)
			r.buf.Write(line.Value(r.src))
		}

		// Close the segment.
		seg.end = r.buf.Len()
		r.append(seg)

		// Close the block.
		r.buf.WriteString("\n---")
		r.endBlock()
	}

	return ast.WalkContinue
}

func (c CodeblockSegment) Bounds() (start, end int) {
	return c.start, c.end
}

func (c CodeblockSegment) CodeblockLanguage() string {
	return c.language
}
