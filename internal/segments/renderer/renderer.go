package renderer

import (
	"bytes"
	"fmt"
	"log"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/state"
	"github.com/diamondburned/cchat-discord/internal/segments/segutil"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/ningen/md"
	"github.com/yuin/goldmark/ast"
)

type Renderer func(r *Text, n ast.Node, enter bool) ast.WalkStatus

var renderers = map[ast.NodeKind]Renderer{}

// Register registers a renderer to a node kind.
func Register(kind ast.NodeKind, r Renderer) {
	log.Printf("Registering kind %v", kind)
	renderers[kind] = r
}

// Parse parses the raw Markdown bytes into a rich text.
func Parse(b []byte) text.Rich {
	node := md.Parse(b)
	return RenderNode(b, node)
}

// RenderNode renders the given raw Markdown bytes with the parsed AST node into
// a rich text.
func RenderNode(source []byte, n ast.Node) text.Rich {
	r := New(source, n)
	r.Walk(n)

	return text.Rich{
		Content:  r.String(),
		Segments: r.Segments,
	}
}

type Text struct {
	Buffer   *bytes.Buffer
	Source   []byte
	Segments []text.Segment
	Inlines  InlineState
	Links    LinkState

	// these fields can be nil
	Message *discord.Message
	Store   state.Store
}

func New(src []byte, node ast.Node) *Text {
	buf := &bytes.Buffer{}
	buf.Grow(len(src))

	return &Text{
		Source:   src,
		Buffer:   buf,
		Segments: make([]text.Segment, 0, node.ChildCount()),
	}
}

func (r *Text) WithState(m *discord.Message, s state.Store) {
	r.Message = m
	r.Store = s
}

// String returns a stringified version of Bytes().
func (r *Text) String() string {
	return string(r.Bytes())
}

// Bytes returns the plain content of the buffer with right spaces trimmed as
// best as it could, that is, the function will not trim right spaces that
// segments use.
func (r *Text) Bytes() []byte {
	// Get the rightmost index out of all the segments.
	var rightmost int
	for _, seg := range r.Segments {
		if _, end := seg.Bounds(); end > rightmost {
			rightmost = end
		}
	}

	// Get the original byte slice.
	org := r.Buffer.Bytes()

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

func (r *Text) WriteStringf(f string, v ...interface{}) (start, end int) {
	return r.WriteString(fmt.Sprintf(f, v...))
}

func (r *Text) WriteString(s string) (start, end int) {
	return segutil.WriteStringBuf(r.Buffer, s)
}

func (r *Text) Write(b []byte) (start, end int) {
	return segutil.WriteBuf(r.Buffer, b)
}

// StartBlock guarantees enough indentation to start a new block.
func (r *Text) StartBlock() {
	r.StartBlockN(2)
}

// EnsureBreak ensures that the current line is a new line.
func (r *Text) EnsureBreak() {
	r.StartBlockN(1)
}

// StartBlockN allows a custom block level.
func (r *Text) StartBlockN(n int) {
	var maxNewlines = 0

	// Peek twice. If the last character is already a new line or we're only at
	// the start of line (length 0), then don't pad.
	if r.Buffer.Len() > 0 {
		for i := 0; i < n; i++ {
			if r.PeekLast(i) != '\n' {
				maxNewlines++
			}
		}
	}

	// Write the padding.
	r.Buffer.Grow(maxNewlines)
	for i := 0; i < maxNewlines; i++ {
		r.Buffer.WriteByte('\n')
	}
}

func (r *Text) EndBlock() {
	// Do the same thing as starting a block.
	r.StartBlock()
}

// Segments returns the previous byte that matches the offset, or 0 if the
// offset goes past the first byte.
func (r *Text) PeekLast(offset int) byte {
	if i := r.Buffer.Len() - offset - 1; i > 0 {
		return r.Buffer.Bytes()[i]
	}
	return 0
}

func (r *Text) Append(segs ...text.Segment) {
	r.Segments = append(r.Segments, segs...)
}

// Clone returns a shallow copy of Text with the new source.
func (r *Text) Clone(src []byte) *Text {
	cpy := *r
	cpy.Source = src
	return &cpy
}

// Join combines the states from renderer with r. Use this with clone.
func (r *Text) Join(renderer *Text) {
	r.Segments = append([]text.Segment(nil), renderer.Segments...)
	r.Inlines = renderer.Inlines.Copy()
}

// Walk walks on the given node with the RenderNode as the walker function.
func (r *Text) Walk(n ast.Node) {
	ast.Walk(n, r.RenderNode)
}

func (r *Text) RenderNode(n ast.Node, enter bool) (ast.WalkStatus, error) {
	f, ok := renderers[n.Kind()]
	if ok {
		return f(r, n, enter), nil
	}

	log.Println("unknown kind:", n.Kind())

	switch n := n.(type) {
	case *ast.Document:
	case *ast.Paragraph:
		if !enter {
			// TODO: investigate
			// r.Buffer.WriteByte('\n')
		}
	case *ast.String:
		if enter {
			r.Buffer.Write(n.Value)
		}
	case *ast.Text:
		if enter {
			r.Buffer.Write(n.Segment.Value(r.Source))

			switch {
			case n.HardLineBreak():
				r.Buffer.WriteString("\n\n")
			case n.SoftLineBreak():
				r.Buffer.WriteByte('\n')
			}
		}
	}

	return ast.WalkContinue, nil
}
