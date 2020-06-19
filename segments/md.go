package segments

import (
	"bytes"
	"strings"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/state"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/ningen/md"
	"github.com/yuin/goldmark/ast"
)

type TextRenderer struct {
	buf  *bytes.Buffer
	src  []byte
	segs []text.Segment
	inls inlineState
}

func ParseMessage(m *discord.Message, s state.Store) text.Rich {
	return ParseWithMessage([]byte(m.Content), s, m, true)
}

func ParseWithMessage(b []byte, s state.Store, m *discord.Message, msg bool) text.Rich {
	node := md.ParseWithMessage(b, s, m, msg)
	return RenderNode(b, node)
}

func Parse(b []byte) text.Rich {
	node := md.Parse(b)
	return RenderNode(b, node)
}

func RenderNode(source []byte, n ast.Node) text.Rich {
	buf := &bytes.Buffer{}
	buf.Grow(len(source))

	r := TextRenderer{
		src:  source,
		buf:  buf,
		segs: make([]text.Segment, 0, n.ChildCount()),
	}

	ast.Walk(n, r.renderNode)

	return text.Rich{
		Content:  buf.String(),
		Segments: r.segs,
	}
}

// i returns the current cursor position.
func (r *TextRenderer) i() int {
	return r.buf.Len()
}

// startBlock guarantees enough indentation to start a new block.
func (r *TextRenderer) startBlock() {
	var maxNewlines = 0

	// Peek twice. If the last character is already a new line or we're only at
	// the start of line (length 0), then don't pad.
	if r.buf.Len() > 0 {
		if r.peekLast(0) != '\n' {
			maxNewlines++
		}
		if r.peekLast(1) != '\n' {
			maxNewlines++
		}
	}

	// Write the padding.
	r.buf.WriteString(strings.Repeat("\n", maxNewlines))
}

func (r *TextRenderer) endBlock() {
	// Do the same thing as starting a block.
	r.startBlock()
}

func (r *TextRenderer) peekLast(offset int) byte {
	if i := r.buf.Len() - offset - 1; i > 0 {
		return r.buf.Bytes()[i]
	}
	return 0
}

func (r *TextRenderer) append(segs ...text.Segment) {
	r.segs = append(r.segs, segs...)
}

func (r *TextRenderer) renderNode(n ast.Node, enter bool) (ast.WalkStatus, error) {
	switch n := n.(type) {
	case *ast.Document:
	case *ast.Paragraph:
		if !enter {
			// TODO: investigate
			// r.buf.WriteByte('\n')
		}
	case *ast.Blockquote:
		return r.blockquote(n, enter), nil
	case *ast.FencedCodeBlock:
		return r.codeblock(n, enter), nil
	case *ast.Link:
		return r.link(n, enter), nil
	case *ast.AutoLink:
		return r.autoLink(n, enter), nil
	case *md.Inline:
		return r.inline(n, enter), nil
	case *md.Emoji:
		return r.emoji(n, enter), nil
	case *md.Mention:
		return r.mention(n, enter), nil
	case *ast.String:
		if enter {
			r.buf.Write(n.Value)
		}
	case *ast.Text:
		if enter {
			r.buf.Write(n.Segment.Value(r.src))

			switch {
			case n.HardLineBreak():
				r.buf.WriteString("\n\n")
			case n.SoftLineBreak():
				r.buf.WriteByte('\n')
			}
		}
	}

	return ast.WalkContinue, nil
}
