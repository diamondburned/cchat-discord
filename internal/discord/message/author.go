package message

import (
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/channel/shared"
	"github.com/diamondburned/cchat-discord/internal/discord/state"
	"github.com/diamondburned/cchat-discord/internal/segments/colored"
	"github.com/diamondburned/cchat-discord/internal/segments/mention"
	"github.com/diamondburned/cchat-discord/internal/segments/reference"
	"github.com/diamondburned/cchat-discord/internal/segments/segutil"
	"github.com/diamondburned/cchat/text"
)

type Author struct {
	name text.Rich
	user *mention.User // same pointer as in name
}

var _ cchat.Author = (*Author)(nil)

// NewAuthor creates a new message author.
func NewAuthor(user *mention.User) Author {
	return Author{
		name: RenderAuthorName(user),
		user: user,
	}
}

// RenderAuthorName renders the given user mention into a text segment.
func RenderAuthorName(user *mention.User) text.Rich {
	var rich text.Rich
	richUser(&rich, user)
	return rich
}

// richMember appends the member name directly into rich.
func richUser(rich *text.Rich, user *mention.User) (start, end int) {
	start, end = segutil.Write(rich, user.DisplayName())

	// Append the bot prefix if the user is a bot.
	if user.User().Bot {
		rich.Content += " "
		rich.Segments = append(rich.Segments,
			colored.NewBlurple(segutil.Write(rich, "[BOT]")),
		)
	}

	rich.Segments = append(rich.Segments, mention.NewSegment(start, end, user))

	return
}

func (a Author) ID() cchat.ID {
	return a.user.UserID().String()
}

func (a Author) Name() text.Rich {
	return a.name
}

func (a Author) Avatar() string {
	return a.user.Avatar()
}

const authorReplyingTo = " replying to "

// AddUserReply modifies Author to make it appear like it's a message reply.
// Specifically, this function is used for direct messages in virtual channels.
func (a *Author) AddUserReply(user discord.User, s *state.Instance) {
	a.name.Content += authorReplyingTo

	userMention := mention.NewUser(user)
	userMention.WithState(s.State)
	userMention.Prefetch()

	richUser(&a.name, userMention)
}

// AddChannelReply adds a reply pointing to a channel. If the given channel is a
// direct message channel, then the first recipient will be used instead, and
// the function will operate similar to AddUserReply.
func (a *Author) AddChannelReply(ch discord.Channel, s *state.Instance) {
	if ch.Type == discord.DirectMessage && len(ch.DMRecipients) > 0 {
		a.AddUserReply(ch.DMRecipients[0], s)
		return
	}

	a.name.Content += authorReplyingTo
	start, end := segutil.Write(&a.name, shared.ChannelName(ch))

	a.name.Segments = append(a.name.Segments,
		mention.Segment{
			Start:   start,
			End:     end,
			Channel: mention.NewChannel(ch),
		},
	)
}

// AddMessageReference adds a message reference to the author.
func (a *Author) AddMessageReference(ref discord.Message, s *state.Instance) {
	a.name.Content += authorReplyingTo

	userMention := mention.NewUser(ref.Author)
	userMention.WithGuildID(ref.GuildID)
	userMention.WithState(s.State)
	userMention.Prefetch()

	start, end := richUser(&a.name, userMention)

	a.name.Segments = append(a.name.Segments,
		reference.NewMessageSegment(start, end, ref.ID),
	)
}
