package inline

import (
	"github.com/diamondburned/cchat-discord/internal/segments/renderer"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/cchat/utils/empty"
	"github.com/diamondburned/ningen/v2/md"
	"github.com/yuin/goldmark/ast"
)

func init() {
	renderer.Register(md.KindInline, inline)
}

// inline parses an inline node. This method at the moment will always create a
// new segment for overlapping attributes.
func inline(r *renderer.Text, node ast.Node, enter bool) ast.WalkStatus {
	n := node.(*md.Inline)
	// For instructions on how this works, refer to inline_attr.jpg.

	// Pop the last segment if it's not empty.
	if !r.Inlines.Empty() {
		r.Inlines.End = r.Buffer.Len()

		// Only use this section if the length is not zero.
		if r.Inlines.Start != r.Inlines.End {
			r.Append(NewSegmentFromState(r.Inlines))
		}
	}

	if enter {
		r.Inlines.Add(n.Attr)
	} else {
		r.Inlines.Remove(n.Attr)
	}

	// Update the start pointer of the current segment.
	r.Inlines.Start = r.Buffer.Len()

	return ast.WalkContinue
}

type Attribute text.Attribute

var _ text.Attributor = (*Attribute)(nil)

func (attr Attribute) Attribute() text.Attribute {
	return text.Attribute(attr)
}

// DimSuffix creates a string with the suffix dimmed.
func DimSuffix(prefix, suffix string) text.Rich {
	return text.Rich{
		Content: prefix + suffix,
		Segments: []text.Segment{
			Segment{
				start:      len(prefix),
				end:        len(prefix) + len(suffix),
				attributes: Attribute(text.AttributeDimmed),
			},
		},
	}
}

func Write(rich *text.Rich, content string, attr text.Attribute) {
	start := len(rich.Content)
	rich.Content += content
	end := len(rich.Content)

	rich.Segments = append(rich.Segments, Segment{
		start:      start,
		end:        end,
		attributes: Attribute(attr),
	})
}

type Segment struct {
	empty.TextSegment
	start, end int
	attributes Attribute
}

// NewSegmentFromState creates a new rich text segment from the renderer's
// inline attribute state.
func NewSegmentFromState(state renderer.InlineState) Segment {
	return NewSegmentFromMD(state.Start, state.End, state.Attributes)
}

// NewSegmentFromMD creates a new rich text segment from the start, end indices
// and the markdown inline attributes.
func NewSegmentFromMD(start, end int, attr md.Attribute) Segment {
	var seg = Segment{
		start: start,
		end:   end,
	}

	if attr.Has(md.AttrBold) {
		seg.attributes |= Attribute(text.AttributeBold)
	}
	if attr.Has(md.AttrItalics) {
		seg.attributes |= Attribute(text.AttributeItalics)
	}
	if attr.Has(md.AttrUnderline) {
		seg.attributes |= Attribute(text.AttributeUnderline)
	}
	if attr.Has(md.AttrStrikethrough) {
		seg.attributes |= Attribute(text.AttributeStrikethrough)
	}
	if attr.Has(md.AttrSpoiler) {
		seg.attributes |= Attribute(text.AttributeSpoiler)
	}
	if attr.Has(md.AttrMonospace) {
		seg.attributes |= Attribute(text.AttributeMonospace)
	}

	return seg
}

func NewSegment(start, end int, attrs ...text.Attribute) Segment {
	var attr = text.AttributeNormal
	for _, a := range attrs {
		attr |= a
	}
	return Segment{
		start:      start,
		end:        end,
		attributes: Attribute(attr),
	}
}

var _ text.Segment = (*Segment)(nil)

func (i Segment) Bounds() (start, end int) {
	return i.start, i.end
}

func (i Segment) AsAttributor() text.Attributor {
	return i.attributes
}
