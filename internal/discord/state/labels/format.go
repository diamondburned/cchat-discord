package labels

import (
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/cchat-discord/internal/segments/avatar"
	"github.com/diamondburned/cchat-discord/internal/segments/colored"
	"github.com/diamondburned/cchat-discord/internal/segments/mention"
	"github.com/diamondburned/cchat-discord/internal/segments/segutil"
	"github.com/diamondburned/cchat-discord/internal/urlutils"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/ningen/v2"
)

// TODO: these functions should probably be its own package.

func labelGuild(s *ningen.State, guildID discord.GuildID) text.Rich {
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

func labelMember(s *ningen.State, g discord.GuildID, u discord.UserID) text.Rich {
	m, err := s.Cabinet.Member(g, u)
	if err != nil {
		s.MemberState.RequestMember(g, u)
		return text.Plain(u.Mention())
	}

	user := mention.NewUser(m.User)
	user.WithMember(*m)
	user.WithGuildID(g)
	user.WithState(s)
	user.Prefetch()

	rich := text.Rich{Content: user.DisplayName()}
	rich.Segments = []text.Segment{
		mention.NewSegment(0, len(rich.Content), user),
	}

	if m.User.Bot {
		rich.Content += " "
		rich.Segments = append(rich.Segments,
			colored.NewBlurple(segutil.Write(&rich, "[BOT]")),
		)
	}

	return rich
}

func labelChannel(s *ningen.State, chID discord.ChannelID) text.Rich {
	var rich text.Rich

	ch, err := s.Cabinet.Channel(chID)
	if err != nil {
		rich = text.Plain(ch.Mention())
	} else {
		rich = text.Plain(mention.ChannelName(*ch))
	}

	return rich
}
