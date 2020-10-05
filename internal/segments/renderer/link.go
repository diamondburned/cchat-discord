package renderer

// LinkState is used for ast.Link segments.
type LinkState struct {
	Linkstack []int // stack of starting integers
}

func (ls *LinkState) Append(l int) {
	ls.Linkstack = append(ls.Linkstack, l)
}

func (ls *LinkState) Pop() int {
	ilast := len(ls.Linkstack) - 1
	start := ls.Linkstack[ilast]
	ls.Linkstack = ls.Linkstack[:ilast]
	return start
}

func (ls LinkState) Len() int {
	return len(ls.Linkstack)
}
