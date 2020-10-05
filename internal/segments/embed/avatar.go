package embed

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/cchat-discord/internal/segments/emoji"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/cchat/utils/empty"
)

type AvatarSegment struct {
	empty.TextSegment
	start int
	url   string
	text  string
	size  int
}

var (
	_ text.Avatarer = (*AvatarSegment)(nil)
	_ text.Segment  = (*AvatarSegment)(nil)
)

func Author(start int, a discord.EmbedAuthor) AvatarSegment {
	return AvatarSegment{
		start: start,
		url:   a.ProxyIcon,
		text:  "Avatar",
	}
}

// Footer uses an avatar segment to comply with Discord.
func Footer(start int, f discord.EmbedFooter) AvatarSegment {
	return AvatarSegment{
		start: start,
		url:   f.ProxyIcon,
		text:  "Icon",
	}
}

func (a AvatarSegment) Bounds() (int, int) {
	return a.start, a.start
}

func (a AvatarSegment) AsAvatarer() text.Avatarer {
	return a
}

// Avatar returns the avatar URL.
func (a AvatarSegment) Avatar() (url string) {
	return a.url
}

// AvatarSize returns the size of a small emoji.
func (a AvatarSegment) AvatarSize() int {
	if a.size > 0 {
		return a.size
	}
	return emoji.InlineSize
}

func (a AvatarSegment) AvatarText() string {
	return a.text
}
