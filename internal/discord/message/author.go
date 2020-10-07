package message

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/state"
	"github.com/diamondburned/cchat-discord/internal/segments/colored"
	"github.com/diamondburned/cchat-discord/internal/segments/mention"
	"github.com/diamondburned/cchat-discord/internal/segments/segutil"
	"github.com/diamondburned/cchat-discord/internal/urlutils"
	"github.com/diamondburned/cchat/text"
)

type Author struct {
	id     discord.UserID
	name   text.Rich
	avatar string
}

var _ cchat.Author = (*Author)(nil)

func NewUser(u discord.User, s *state.Instance) Author {
	var name = text.Rich{Content: u.Username}
	if u.Bot {
		name.Content += " "
		name.Segments = append(name.Segments,
			colored.NewBlurple(segutil.Write(&name, "[BOT]")),
		)
	}

	// Append a clickable user popup.
	useg := mention.UserSegment(0, len(name.Content), u)
	useg.WithState(s.State)
	name.Segments = append(name.Segments, useg)

	return Author{
		id:     u.ID,
		name:   name,
		avatar: urlutils.AvatarURL(u.AvatarURL()),
	}
}

func NewGuildMember(m discord.Member, g discord.Guild, s *state.Instance) Author {
	return Author{
		id:     m.User.ID,
		name:   RenderMemberName(m, g, s),
		avatar: urlutils.AvatarURL(m.User.AvatarURL()),
	}
}

func RenderMemberName(m discord.Member, g discord.Guild, s *state.Instance) text.Rich {
	var name = text.Rich{
		Content: m.User.Username,
	}

	// Update the nickname.
	if m.Nick != "" {
		name.Content = m.Nick
	}

	// Update the color.
	if c := discord.MemberColor(g, m); c > 0 {
		name.Segments = append(name.Segments,
			colored.New(len(name.Content), c.Uint32()),
		)
	}

	// Append the bot prefix if the user is a bot.
	if m.User.Bot {
		name.Content += " "
		name.Segments = append(name.Segments,
			colored.NewBlurple(segutil.Write(&name, "[BOT]")),
		)
	}

	// Append a clickable user popup.
	useg := mention.MemberSegment(0, len(name.Content), g, m)
	useg.WithState(s.State)
	name.Segments = append(name.Segments, useg)

	return name
}

func (a Author) ID() cchat.ID {
	return a.id.String()
}

func (a Author) Name() text.Rich {
	return a.name
}

func (a Author) Avatar() string {
	return a.avatar
}
