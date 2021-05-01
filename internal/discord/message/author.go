package message

import (
	"context"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/state"
	"github.com/diamondburned/cchat-discord/internal/segments/colored"
	"github.com/diamondburned/cchat-discord/internal/segments/mention"
	"github.com/diamondburned/cchat-discord/internal/segments/reference"
	"github.com/diamondburned/cchat-discord/internal/segments/segutil"
	"github.com/diamondburned/cchat/text"
)

type Author struct {
	name  text.Rich
	user  *mention.User // same pointer as in name
	state *state.Instance
}

var _ cchat.User = (*Author)(nil)

// NewAuthor creates a new message author.
func NewAuthor(s *state.Instance, user *mention.User) Author {
	user.WithState(s.State)

	return Author{
		name:  RenderUserName(user),
		user:  user,
		state: s,
	}
}

// RenderUserName renders the given user mention into a text segment.
func RenderUserName(user *mention.User) text.Rich {
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

	rich.Segments = append(rich.Segments, mention.Segment{
		Start: start,
		End:   end,
		User:  user,
	})

	return
}

// ID returns the author's ID. The ID may not be a valid Discord user ID if the
// user is not a valid (real) user (e.g. webhooks).
func (a Author) ID() cchat.ID {
	user := a.user.User()

	id := user.ID.String()

	// Treat pseudo-users specially.
	if user.Discriminator == "0000" {
		id += "_" + user.Username
	}

	return id
}

// Name subscribes the author to the global name label registry.
func (a Author) Name(_ context.Context, l cchat.LabelContainer) (func(), error) {
	l.SetLabel(a.name)
	return func() {}, nil
}

const authorReplyingTo = " replying to "

// AddUserReply modifies User to make it appear like it's a message reply.
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
	start, end := segutil.Write(&a.name, mention.ChannelName(ch))

	a.name.Segments = append(a.name.Segments,
		mention.Segment{
			Start:   start,
			End:     end,
			Channel: mention.NewChannel(ch),
		},
	)
}

// AddMessageReference adds a message reference to the user.
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
