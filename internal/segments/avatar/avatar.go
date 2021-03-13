package avatar

import (
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/cchat/utils/empty"
)

// Segment describes an avatar segment.
type Segment struct {
	empty.TextSegment
	Position int

	URL  string
	Size int    // optional
	Text string // optional
}

func (s Segment) Bounds() (int, int)        { return s.Position, s.Position }
func (s Segment) AsAvatarer() text.Avatarer { return avatarURL{s} }

type avatarURL struct {
	seg Segment
}

var _ text.Avatarer = avatarURL{}

func (aurl avatarURL) AvatarText() string {
	return aurl.seg.Text
}

func (aurl avatarURL) AvatarSize() int {
	return aurl.seg.Size
}

func (aurl avatarURL) Avatar() string {
	return aurl.seg.URL
}