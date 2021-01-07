package segments

import (
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/state/store"
	"github.com/diamondburned/cchat-discord/internal/segments/embed"
	"github.com/diamondburned/cchat-discord/internal/segments/renderer"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/ningen/v2/md"

	_ "github.com/diamondburned/cchat-discord/internal/segments/blockquote"
	_ "github.com/diamondburned/cchat-discord/internal/segments/codeblock"
	_ "github.com/diamondburned/cchat-discord/internal/segments/colored"
	_ "github.com/diamondburned/cchat-discord/internal/segments/emoji"
	_ "github.com/diamondburned/cchat-discord/internal/segments/inline"
	_ "github.com/diamondburned/cchat-discord/internal/segments/link"
	_ "github.com/diamondburned/cchat-discord/internal/segments/mention"
)

func ParseMessage(m *discord.Message, s store.Cabinet) text.Rich {
	var rich text.Rich
	ParseMessageRich(&rich, m, s)
	return rich
}

func ParseMessageRich(rich *text.Rich, m *discord.Message, s store.Cabinet) {
	content := []byte(m.Content)

	r := renderer.New(content)
	r.Buffer.Grow(len(rich.Content))
	r.Buffer.WriteString(rich.Content)

	// Register the needed states for some renderers.
	r.WithState(m, s)

	// Render the main message body.
	if len(content) > 0 {
		node := md.ParseWithMessage(content, s, m, true)
		r.Walk(node)
	}

	// Render the extra bits.
	embed.RenderAttachments(r, m.Attachments)
	embed.RenderEmbeds(r, m.Embeds, m, s)

	rich.Content = r.String()
	rich.Segments = append(rich.Segments, r.Segments...)
}

func ParseWithMessage(b []byte, m *discord.Message, s store.Cabinet) text.Rich {
	var rich text.Rich
	ParseWithMessageRich(&rich, b, m, s)
	return rich
}

func ParseWithMessageRich(rich *text.Rich, b []byte, m *discord.Message, s store.Cabinet) {
	if len(b) == 0 {
		return
	}

	node := md.ParseWithMessage(b, s, m, true)

	r := renderer.New(b)
	r.Buffer.Grow(len(rich.Content))
	r.Buffer.WriteString(rich.Content)

	r.WithState(m, s)
	r.Walk(node)

	rich.Content = r.String()
	rich.Segments = append(rich.Segments, r.Segments...)
}

// Ellipsize caps the length of the rendered text segment to be not longer than
// the given length. The ellipsize will be appended if it is.
func Ellipsize(rich text.Rich, maxLen int) text.Rich {
	ellipsize := maxLen < len(rich.Content)
	if !ellipsize {
		maxLen = len(rich.Content)
	}

	substr := Substring(rich, 0, maxLen)
	if ellipsize {
		substr.Content += "…"
	}

	return substr
}

// Substring slices the given rich text.
func Substring(rich text.Rich, start, end int) text.Rich {
	substring := text.Rich{
		Content:  rich.Content[start:end],
		Segments: make([]text.Segment, 0, len(rich.Segments)),
	}

	for _, seg := range rich.Segments {
		i, j := seg.Bounds()

		// Bound-check: check if the starting point is within the range.
		if start <= i && i <= end {
			// If the current segment is cleanly within the bound, then we can
			// directly insert it.
			if j <= end {
				substring.Segments = append(substring.Segments, seg)
				continue
			}

			substring.Segments = append(substring.Segments, trimmedSegment{
				Segment: seg,
				start:   i, // preserve the segment's starting point
				end:     end,
			})
		}
	}

	return substring
}

type trimmedSegment struct {
	text.Segment
	start, end int
}

func (seg trimmedSegment) Bounds() (int, int) {
	return seg.start, seg.end
}
