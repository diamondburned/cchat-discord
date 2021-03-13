package mention

import (
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/cchat-discord/internal/segments/avatar"
	"github.com/diamondburned/cchat-discord/internal/urlutils"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/ningen/v2"
)

// NewGuildText creates a new rich text describing the given member fetched from
// the state.
func NewGuildText(s *ningen.State, guildID discord.GuildID) text.Rich {
	g, err := s.Cabinet.Guild(guildID)
	if err != nil {
		return text.Plain(guildID.String())
	}

	return text.Rich{
		Content: g.Name,
		Segments: []text.Segment{
			avatar.Segment{
				URL:  urlutils.AvatarURL(g.IconURL()),
				Size: urlutils.AvatarSize,
				Text: g.Name,
			},
		},
	}
}
