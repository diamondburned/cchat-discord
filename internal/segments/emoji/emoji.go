package emoji

import (
	"net/url"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/cchat-discord/internal/segments/renderer"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/cchat/utils/empty"
	"github.com/diamondburned/ningen/md"
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
	u, err := url.Parse(fullURL)
	if err != nil {
		return fullURL
	}

	v := u.Query()
	v.Set("size", "64")

	u.RawQuery = v.Encode()
	return u.String()
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
		EmojiURL: injectSizeURL(e.EmojiURL()),
		Large:    large,
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
