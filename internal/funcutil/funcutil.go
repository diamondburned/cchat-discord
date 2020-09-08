package funcutil

// NweCancels creates a stateful closure for adding callbacks into a list.
func NewCancels() func(...func()) []func() {
	var cancels []func()
	return func(appended ...func()) []func() {
		cancels = append(cancels, appended...)
		return cancels
	}
}

// JoinCancels joins multiple cancel callbacks into one.
func JoinCancels(cancellers []func()) func() {
	return func() {
		for _, c := range cancellers {
			c()
		}
	}
}
