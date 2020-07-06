package urlutils

import (
	"net/url"
	"path"
	"strconv"
	"strings"
)

// Sized wraps the URL with the size query.
func Sized(URL string, size int) string {
	u, err := url.Parse(URL)
	if err != nil {
		return URL
	}

	q := u.Query()
	q.Set("size", strconv.Itoa(size))
	u.RawQuery = q.Encode()

	return u.String()
}

// Ext returns the lowercased file extension of the URL.
func Ext(URL string) string {
	u, err := url.Parse(URL)
	if err != nil {
		return ""
	}

	return strings.ToLower(path.Ext(u.Path))
}

func ExtIs(URL string, exts []string) bool {
	var ext = Ext(URL)

	for _, e := range exts {
		if e == ext {
			return true
		}
	}

	return false
}
