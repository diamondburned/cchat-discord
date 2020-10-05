package renderer

import (
	"github.com/diamondburned/ningen/md"
)

// InlineState assists in keeping a stateful inline segment builder.
type InlineState struct {
	// TODO: use a stack to allow overlapping
	Start, End int
	Attributes md.Attribute
}

func (i *InlineState) Add(attr md.Attribute) {
	i.Attributes |= attr
}

func (i *InlineState) Remove(attr md.Attribute) {
	i.Attributes &= ^attr
}

func (i InlineState) Copy() InlineState {
	return i
}

func (i InlineState) Empty() bool {
	return i.Attributes == 0 || i.Start < i.End
}
