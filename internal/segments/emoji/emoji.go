package emoji

import (
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/cchat-discord/internal/segments/renderer"
	"github.com/diamondburned/cchat-discord/internal/urlutils"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/cchat/utils/empty"
	"github.com/diamondburned/ningen/v2/md"
	"github.com/yuin/goldmark/ast"
)

func init() {
	renderer.Register(md.KindEmoji, emoji)
}

func emoji(r *renderer.Text, node ast.Node, enter bool) ast.WalkStatus {
	n := node.(*md.Emoji)

	if enter {
		r.Append(Segment{
			Start: r.Buffer.Len(),
			Emoji: EmojiFromNode(n),
		})
	}

	return ast.WalkContinue
}

const (
	InlineSize = 22
	LargeSize  = 48
)

type Emoji struct {
	Name     string
	EmojiURL string
	Large    bool
}

var _ text.Imager = (*Emoji)(nil)

func injectSizeURL(fullURL string) string {
	return urlutils.Sized(fullURL, 64)
}

func EmojiFromNode(n *md.Emoji) Emoji {
	return Emoji{
		Name:     n.Name,
		Large:    n.Large,
		EmojiURL: injectSizeURL(n.EmojiURL()),
	}
}

func EmojiFromDiscord(e discord.Emoji, large bool) Emoji {
	return Emoji{
		Name:     e.Name,
		Large:    large,
		EmojiURL: injectSizeURL(e.EmojiURL()),
	}
}

func (e Emoji) Image() string {
	return e.EmojiURL
}

func (e Emoji) ImageSize() (w, h int) {
	if e.Large {
		return LargeSize, LargeSize
	}
	return InlineSize, InlineSize
}

func (e Emoji) ImageText() string {
	return ":" + e.Name + ":"
}

type Segment struct {
	empty.TextSegment
	Start int
	Emoji Emoji
}

var _ text.Segment = (*Segment)(nil)

func (e Segment) Bounds() (start, end int) {
	return e.Start, e.Start
}

func (e Segment) AsImager() text.Imager { return e.Emoji }
