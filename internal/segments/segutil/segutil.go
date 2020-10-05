package segutil

import (
	"bytes"

	"github.com/diamondburned/cchat/text"
)

// helper global functions

func WriteBuf(w *bytes.Buffer, b []byte) (start, end int) {
	start = w.Len()
	w.Write(b)
	end = w.Len()
	return start, end
}

func WriteStringBuf(w *bytes.Buffer, b string) (start, end int) {
	start = w.Len()
	w.WriteString(b)
	end = w.Len()
	return start, end
}

func Add(r *text.Rich, seg ...text.Segment) {
	r.Segments = append(r.Segments, seg...)
}
