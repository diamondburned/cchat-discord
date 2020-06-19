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
		// Create a segment.
		seg := CodeblockSegment{
			start:    r.i(),
			language: string(n.Language(r.src)),
		}

		// Join all lines together.
		var lines = n.Lines()

		for i := 0; i < lines.Len(); i++ {
			line := lines.At(i)
			r.buf.Write(line.Value(r.src))
		}

		// Close the segment.
		seg.end = r.i()
		r.append(seg)
	}

	return ast.WalkContinue
}

func (c CodeblockSegment) Bounds() (start, end int) {
	return c.start, c.end
}

func (c CodeblockSegment) CodeblockLanguage() string {
	return c.language
}
