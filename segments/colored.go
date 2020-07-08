package segments

import "github.com/diamondburned/cchat/text"

type Colored struct {
	start int
	end   int
	color uint32
}

var (
	_ text.Colorer = (*Colored)(nil)
	_ text.Segment = (*Colored)(nil)
)

func NewColored(strlen int, color uint32) Colored {
	return Colored{0, strlen, color}
}

func NewColoredSegment(start, end int, color uint32) Colored {
	return Colored{start, end, color}
}

func (color Colored) Bounds() (start, end int) {
	return color.start, color.end
}

func (color Colored) Color() uint32 {
	return color.color
}
