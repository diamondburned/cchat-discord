package embed

import (
	"fmt"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/cchat-discord/internal/urlutils"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/cchat/utils/empty"
	"github.com/dustin/go-humanize"
)

type ImageSegment struct {
	empty.TextSegment
	start int
	url   string
	w, h  int
	text  string
}

var (
	_ text.Imager  = (*ImageSegment)(nil)
	_ text.Segment = (*ImageSegment)(nil)
)

func Image(start int, i discord.EmbedImage) ImageSegment {
	return ImageSegment{
		start: start,
		url:   i.Proxy,
		w:     int(i.Width),
		h:     int(i.Height),
		text:  fmt.Sprintf("Image (%s)", urlutils.Name(i.URL)),
	}
}

func Thumbnail(start int, t discord.EmbedThumbnail) ImageSegment {
	return ImageSegment{
		start: start,
		url:   t.Proxy,
		w:     int(t.Width),
		h:     int(t.Height),
		text:  fmt.Sprintf("Thumbnail (%s)", urlutils.Name(t.URL)),
	}
}

func Attachment(start int, a discord.Attachment) ImageSegment {
	return ImageSegment{
		start: start,
		url:   a.Proxy,
		w:     int(a.Width),
		h:     int(a.Height),
		text:  fmt.Sprintf("%s (%s)", a.Filename, humanize.Bytes(a.Size)),
	}
}

func (i ImageSegment) Bounds() (start, end int) {
	return i.start, i.start
}

func (i ImageSegment) AsImager() text.Imager {
	return i
}

// Image returns the URL.
func (i ImageSegment) Image() string {
	return i.url
}

func (i ImageSegment) ImageSize() (w, h int) {
	return i.w, i.h
}

func (i ImageSegment) ImageText() string {
	return i.text
}
