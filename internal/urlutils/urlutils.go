package urlutils

import (
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/diamondburned/arikawa/discord"
)

// AvatarURL wraps the URL with URL queries for the avatar.
func AvatarURL(URL string) string {
	return Sized(URL, 64)
}

// Sized wraps the URL with the size query.
func Sized(URL string, size int) string {
	if URL == "" {
		return ""
	}

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

func Name(URL string) string {
	u, err := url.Parse(URL)
	if err != nil {
		return URL
	}
	return path.Base(u.Path)
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

// AssetURL generates the image URL from the given asset image ID.
func AssetURL(appID discord.AppID, imageID string) string {
	if strings.HasPrefix(imageID, "spotify:") {
		return "https://i.scdn.co/image/" + strings.TrimPrefix(imageID, "spotify:")
	}
	return "https://cdn.discordapp.com/app-assets/" + appID.String() + "/" + imageID + ".png"
}
