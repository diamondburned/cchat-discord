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
	start    int
	name     string
	emojiURL string
	large    bool
}

var _ text.Imager = (*EmojiSegment)(nil)

func (r *TextRenderer) emoji(n *md.Emoji, enter bool) ast.WalkStatus {
	if enter {
		r.append(EmojiSegment{
			start:    r.i(),
			name:     n.Name,
			large:    n.Large,
			emojiURL: n.EmojiURL() + "&size=64",
		})
	}

	return ast.WalkContinue
}

func (e EmojiSegment) Bounds() (start, end int) {
	return e.start, e.start
}

func (e EmojiSegment) Image() string {
	return e.emojiURL
}

// TODO: large emoji

func (e EmojiSegment) ImageSize() (w, h int) {
	if e.large {
		return LargeEmojiSize, LargeEmojiSize
	}
	return InlineEmojiSize, InlineEmojiSize
}

func (e EmojiSegment) ImageText() string {
	return ":" + e.name + ":"
}
