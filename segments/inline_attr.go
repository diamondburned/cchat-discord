package segments

import (
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/ningen/md"
	"github.com/yuin/goldmark/ast"
)

type inlineState struct {
	// TODO: use a stack to allow overlapping
	InlineSegment
}

func (i *inlineState) add(attr md.Attribute) {
	if attr.Has(md.AttrBold) {
		i.attributes |= text.AttrBold
	}
	if attr.Has(md.AttrItalics) {
		i.attributes |= text.AttrItalics
	}
	if attr.Has(md.AttrUnderline) {
		i.attributes |= text.AttrUnderline
	}
	if attr.Has(md.AttrStrikethrough) {
		i.attributes |= text.AttrStrikethrough
	}
	if attr.Has(md.AttrSpoiler) {
		i.attributes |= text.AttrSpoiler
	}
	if attr.Has(md.AttrMonospace) {
		i.attributes |= text.AttrMonospace
	}
}

func (i *inlineState) remove(attr md.Attribute) {
	if attr.Has(md.AttrBold) {
		i.attributes &= ^text.AttrBold
	}
	if attr.Has(md.AttrItalics) {
		i.attributes &= ^text.AttrItalics
	}
	if attr.Has(md.AttrUnderline) {
		i.attributes &= ^text.AttrUnderline
	}
	if attr.Has(md.AttrStrikethrough) {
		i.attributes &= ^text.AttrStrikethrough
	}
	if attr.Has(md.AttrSpoiler) {
		i.attributes &= ^text.AttrSpoiler
	}
	if attr.Has(md.AttrMonospace) {
		i.attributes &= ^text.AttrMonospace
	}
}

func (i inlineState) copy() InlineSegment {
	return i.InlineSegment
}

type InlineSegment struct {
	start, end int
	attributes text.Attribute
}

var _ text.Attributor = (*InlineSegment)(nil)

// inline parses an inline node. This method at the moment will always create a
// new segment for overlapping attributes.
func (r *TextRenderer) inline(n *md.Inline, enter bool) ast.WalkStatus {
	// For instructions on how this works, refer to inline_attr.jpg.

	// Pop the last segment if it's not empty.
	if !r.inls.empty() {
		r.inls.end = r.i()

		// Only use this section if the length is not zero.
		if r.inls.start != r.inls.end {
			r.append(r.inls.copy())
		}
	}

	if enter {
		r.inls.add(n.Attr)
	} else {
		r.inls.remove(n.Attr)
	}

	// Update the start pointer of the current segment.
	r.inls.start = r.i()

	return ast.WalkContinue
}

func (i InlineSegment) Bounds() (start, end int) {
	return i.start, i.end
}

func (i InlineSegment) Attribute() text.Attribute {
	return i.attributes
}

func (i InlineSegment) empty() bool {
	return i.attributes == 0 || i.start < i.end
}
