package segments

import (
	"bytes"
	"fmt"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/state"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/ningen/md"
	"github.com/yuin/goldmark/ast"
)

func ParseMessage(m *discord.Message, s state.Store) text.Rich {
	var content = []byte(m.Content)
	var node = md.ParseWithMessage(content, s, m, true)

	r := NewTextReader(content, node)
	// Register the needed states for some renderers.
	r.WithState(m, s)
	// Render the main message body.
	r.walk(node)
	// Render the extra bits.
	r.renderAttachments(m.Attachments)
	r.renderEmbeds(m.Embeds, m, s)

	return text.Rich{
		Content:  r.String(),
		Segments: r.segs,
	}
}

func ParseWithMessage(b []byte, m *discord.Message, s state.Store, msg bool) text.Rich {
	node := md.ParseWithMessage(b, s, m, msg)
	return RenderNode(b, node)
}

func Parse(b []byte) text.Rich {
	node := md.Parse(b)
	return RenderNode(b, node)
}

func RenderNode(source []byte, n ast.Node) text.Rich {
	r := NewTextReader(source, n)
	r.walk(n)

	return text.Rich{
		Content:  r.String(),
		Segments: r.segs,
	}
}

type TextRenderer struct {
	buf  *bytes.Buffer
	src  []byte
	segs []text.Segment
	inls inlineState

	// these fields can be nil
	msg   *discord.Message
	store state.Store
}

func NewTextReader(src []byte, node ast.Node) TextRenderer {
	buf := &bytes.Buffer{}
	buf.Grow(len(src))

	return TextRenderer{
		src:  src,
		buf:  buf,
		segs: make([]text.Segment, 0, node.ChildCount()),
	}
}

func (r *TextRenderer) WithState(m *discord.Message, s state.Store) {
	r.msg = m
	r.store = s
}

// String returns a stringified version of Bytes().
func (r *TextRenderer) String() string {
	return string(r.Bytes())
}

// Bytes returns the plain content of the buffer with right spaces trimmed as
// best as it could, that is, the function will not trim right spaces that
// segments use.
func (r *TextRenderer) Bytes() []byte {
	// Get the rightmost index out of all the segments.
	var rightmost int
	for _, seg := range r.segs {
		if _, end := seg.Bounds(); end > rightmost {
			rightmost = end
		}
	}

	// Get the original byte slice.
	org := r.buf.Bytes()

	// Trim the right spaces.
	trbuf := bytes.TrimRight(org, "\n")

	// If we trimmed way too far, then slice so that we get as far as the
	// rightmost segment.
	if len(trbuf) < rightmost {
		return org[:rightmost]
	}

	// Else, we're safe returning the trimmed slice.
	return trbuf
}

func (r *TextRenderer) writeStringf(f string, v ...interface{}) (start, end int) {
	return r.writeString(fmt.Sprintf(f, v...))
}

func (r *TextRenderer) writeString(s string) (start, end int) {
	return writestringbuf(r.buf, s)
}

func (r *TextRenderer) write(b []byte) (start, end int) {
	return writebuf(r.buf, b)
}

// startBlock guarantees enough indentation to start a new block.
func (r *TextRenderer) startBlock() {
	r.startBlockN(2)
}

// ensureBreak ensures that the current line is a new line.
func (r *TextRenderer) ensureBreak() {
	r.startBlockN(1)
}

// startBlockN allows a custom block level.
func (r *TextRenderer) startBlockN(n int) {
	var maxNewlines = 0

	// Peek twice. If the last character is already a new line or we're only at
	// the start of line (length 0), then don't pad.
	if r.buf.Len() > 0 {
		for i := 0; i < n; i++ {
			if r.peekLast(i) != '\n' {
				maxNewlines++
			}
		}
	}

	// Write the padding.
	r.buf.Grow(maxNewlines)
	for i := 0; i < maxNewlines; i++ {
		r.buf.WriteByte('\n')
	}
}

func (r *TextRenderer) endBlock() {
	// Do the same thing as starting a block.
	r.startBlock()
}

// peekLast returns the previous byte that matches the offset, or 0 if the
// offset goes past the first byte.
func (r *TextRenderer) peekLast(offset int) byte {
	if i := r.buf.Len() - offset - 1; i > 0 {
		return r.buf.Bytes()[i]
	}
	return 0
}

func (r *TextRenderer) append(segs ...text.Segment) {
	r.segs = append(r.segs, segs...)
}

// clone returns a shallow copy of TextRenderer with the new source.
func (r *TextRenderer) clone(src []byte) *TextRenderer {
	cpy := *r
	cpy.src = src
	return &cpy
}

// join combines the states from renderer with r. Use this with clone.
func (r *TextRenderer) join(renderer *TextRenderer) {
	r.segs = renderer.segs
	r.inls = renderer.inls
}

func (r *TextRenderer) walk(n ast.Node) {
	ast.Walk(n, r.renderNode)
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

// helper global functions

func writebuf(w *bytes.Buffer, b []byte) (start, end int) {
	start = w.Len()
	w.Write(b)
	end = w.Len()
	return start, end
}

func writestringbuf(w *bytes.Buffer, b string) (start, end int) {
	start = w.Len()
	w.WriteString(b)
	end = w.Len()
	return start, end
}

func segmentadd(r *text.Rich, seg ...text.Segment) {
	r.Segments = append(r.Segments, seg...)
}
