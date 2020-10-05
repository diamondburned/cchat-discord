package colored

import (
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/cchat/utils/empty"
)

const Blurple = 0x7289DAFF

type Color uint32

var _ text.Colorer = (*Color)(nil)

// FromRGB converts the 24-bit RGB color to 32-bit RGBA.
func FromRGB(rgb uint32) Color {
	return Color(text.SolidColor(rgb))
}

func (c Color) Color() uint32 {
	return uint32(c)
}

// Segment implements a colored text segment.
type Segment struct {
	empty.TextSegment
	start int
	end   int
	color Color
}

var _ text.Segment = (*Segment)(nil)

func New(strlen int, color uint32) Segment {
	return NewSegment(0, strlen, color)
}

func NewBlurple(start, end int) Segment {
	return Segment{
		start: start,
		end:   end,
		color: Blurple,
	}
}

func NewSegment(start, end int, color uint32) Segment {
	return Segment{
		start: start,
		end:   end,
		color: FromRGB(color),
	}
}

func (seg Segment) Bounds() (start, end int) {
	return seg.start, seg.end
}

func (seg Segment) AsColorer() text.Colorer {
	return seg.color
}
