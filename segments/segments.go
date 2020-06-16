package segments

import "github.com/diamondburned/cchat/text"

type Colored struct {
	strlen int
	color  uint32
}

var (
	_ text.Colorer = (*Colored)(nil)
	_ text.Segment = (*Colored)(nil)
)

func NewColored(strlen int, color uint32) Colored {
	return Colored{strlen, color}
}

func (color Colored) Bounds() (start, end int) {
	return 0, color.strlen
}

func (color Colored) Color() uint32 {
	return color.color
}
