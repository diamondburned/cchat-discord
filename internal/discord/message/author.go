package message

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/state"
	"github.com/diamondburned/cchat-discord/internal/segments/colored"
	"github.com/diamondburned/cchat-discord/internal/segments/mention"
	"github.com/diamondburned/cchat-discord/internal/segments/reference"
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
	var rich text.Rich
	richUser(&rich, u, s)

	return Author{
		id:     u.ID,
		name:   rich,
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
	var rich text.Rich
	richMember(&rich, m, g, s)
	return rich
}

// richMember appends the member name directly into rich.
func richMember(
	rich *text.Rich, m discord.Member, g discord.Guild, s *state.Instance) (start, end int) {

	var displayName = m.User.Username
	if m.Nick != "" {
		displayName = m.Nick
	}

	start, end = segutil.Write(rich, displayName)

	// Update the color.
	if c := discord.MemberColor(g, m); c > 0 {
		rich.Segments = append(rich.Segments,
			colored.NewSegment(start, end, c.Uint32()),
		)
	}

	// Append the bot prefix if the user is a bot.
	if m.User.Bot {
		rich.Content += " "
		rich.Segments = append(rich.Segments,
			colored.NewBlurple(segutil.Write(rich, "[BOT]")),
		)
	}

	// Append a clickable user popup.
	useg := mention.MemberSegment(start, end, g, m)
	useg.WithState(s.State)
	rich.Segments = append(rich.Segments, useg)

	return
}

func richUser(
	rich *text.Rich, u discord.User, s *state.Instance) (start, end int) {

	start, end = segutil.Write(rich, u.Username)

	// Append the bot prefix if the user is a bot.
	if u.Bot {
		rich.Content += " "
		rich.Segments = append(rich.Segments,
			colored.NewBlurple(segutil.Write(rich, "[BOT]")),
		)
	}

	// Append a clickable user popup.
	useg := mention.UserSegment(start, end, u)
	useg.WithState(s.State)
	rich.Segments = append(rich.Segments, useg)

	return
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

const authorReplyingTo = " replying to "

// AddUserReply modifies Author to make it appear like it's a message reply.
// Specifically, this function is used for direct messages.
func (a *Author) AddUserReply(user discord.User, s *state.Instance) {
	a.name.Content += authorReplyingTo
	richUser(&a.name, user, s)
}

func (a *Author) AddReply(name string) {
	a.name.Content += authorReplyingTo + name
}

// // AddMemberReply modifies Author to make it appear like it's a message reply.
// // Specifically, this function is used for guild messages.
// func (a *Author) AddMemberReply(m discord.Member, g discord.Guild, s *state.Instance) {
// 	a.name.Content += authorReplyingTo
// 	richMember(&a.name, m, g, s)
// }

// AddMessageReference adds a message reference to the author.
func (a *Author) AddMessageReference(msgref discord.Message, s *state.Instance) {
	if !msgref.GuildID.IsValid() {
		a.name.Content += authorReplyingTo
		start, end := richUser(&a.name, msgref.Author, s)

		a.name.Segments = append(a.name.Segments,
			reference.NewMessageSegment(start, end, msgref.ID),
		)
		return
	}

	g, err := s.Guild(msgref.GuildID)
	if err != nil {
		return
	}

	m, err := s.Member(g.ID, msgref.Author.ID)
	if err != nil {
		return
	}

	a.name.Content += authorReplyingTo
	start, end := richMember(&a.name, *m, *g, s)

	a.name.Segments = append(a.name.Segments,
		reference.NewMessageSegment(start, end, msgref.ID),
	)
}
