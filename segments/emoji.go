package segments

import (
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/ningen/md"
	"github.com/yuin/goldmark/ast"
)

const (
	InlineEmojiSize = 22
	LargeEmojiSize  = 48
)

type EmojiSegment struct {
	Start    int
	Name     string
	EmojiURL string
	Large    bool
}

var _ text.Imager = (*EmojiSegment)(nil)

func (r *TextRenderer) emoji(n *md.Emoji, enter bool) ast.WalkStatus {
	if enter {
		r.append(EmojiSegment{
			Start:    r.buf.Len(),
			Name:     n.Name,
			Large:    n.Large,
			EmojiURL: n.EmojiURL() + "&size=64",
		})
	}

	return ast.WalkContinue
}

func (e EmojiSegment) Bounds() (start, end int) {
	return e.Start, e.Start
}

func (e EmojiSegment) Image() string {
	return e.EmojiURL
}

// TODO: large emoji

func (e EmojiSegment) ImageSize() (w, h int) {
	if e.Large {
		return LargeEmojiSize, LargeEmojiSize
	}
	return InlineEmojiSize, InlineEmojiSize
}

func (e EmojiSegment) ImageText() string {
	return ":" + e.Name + ":"
}
