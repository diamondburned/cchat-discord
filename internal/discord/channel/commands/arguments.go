package commands

import (
	"bytes"
	"strings"
)

type Arguments []string

func (args Arguments) writeHelp(builder *bytes.Buffer) {
	for i, arg := range args {
		builder.WriteByte(' ')

		// Always treat the last argument as a must.
		if i == len(args)-1 {
			builder.WriteByte('<')
			builder.WriteString(arg)
			builder.WriteByte('>')
		} else {
			builder.WriteByte('[')
			builder.WriteString(arg)
			builder.WriteByte(']')
		}
	}
}

// At returns a two-part string if i is in the list of arguments. Two empty
// strings are returned if i is out of bounds. If the argument is not a flag
// (i.e. not optional), then flag is empty, but name isn't.
func (args Arguments) At(i int) (name, flag string) {
	if i >= len(args) {
		return "", ""
	}

	arg := args[i]
	fis := strings.Fields(arg)

	if len(fis) != 2 {
		return arg, ""
	}

	return fis[1], fis[0]
}
