package discord

import (
	"context"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/segments"
	"github.com/diamondburned/cchat/text"
	"github.com/pkg/errors"
)

type Channel struct {
	id      discord.Snowflake
	guildID discord.Snowflake
	name    string
	session *Session
}

var (
	_ cchat.Server              = (*Channel)(nil)
	_ cchat.ServerMessage       = (*Channel)(nil)
	_ cchat.ServerMessageSender = (*Channel)(nil)
	// _ cchat.ServerMessageSendCompleter = (*Channel)(nil)
	_ cchat.ServerNickname = (*Channel)(nil)
	// _ cchat.ServerMessageEditor        = (*Channel)(nil)
	// _ cchat.ServerMessageActioner      = (*Channel)(nil)
)

func NewChannel(s *Session, ch discord.Channel) *Channel {
	return &Channel{
		id:      ch.ID,
		guildID: ch.GuildID,
		name:    ch.Name,
		session: s,
	}
}

func (ch *Channel) ID() string {
	return ch.id.String()
}

func (ch *Channel) Name() text.Rich {
	return text.Rich{Content: "#" + ch.name}
}

func (ch *Channel) Nickname(ctx context.Context, labeler cchat.LabelContainer) error {
	// We don't have a nickname if we're not in a guild.
	if !ch.guildID.Valid() {
		return nil
	}

	state := ch.session.WithContext(ctx)

	// MemberColor should fill up the state cache.
	c, err := state.MemberColor(ch.guildID, ch.session.userID)
	if err != nil {
		return errors.Wrap(err, "Failed to get self member color")
	}

	m, err := state.Member(ch.guildID, ch.session.userID)
	if err != nil {
		return errors.Wrap(err, "Failed to get self member")
	}

	var rich = text.Rich{Content: m.User.Username}
	if m.Nick != "" {
		rich.Content = m.Nick
	}
	if c > 0 {
		rich.Segments = []text.Segment{
			segments.NewColored(len(rich.Content), c.Uint32()),
		}
	}

	labeler.SetLabel(rich)
	return nil
}

func (ch *Channel) JoinServer(ctx context.Context, ct cchat.MessagesContainer) (func(), error) {
	state := ch.session.WithContext(ctx)

	m, err := state.Messages(ch.id)
	if err != nil {
		return nil, err
	}

	var addcancel = newCancels()

	if ch.guildID.Valid() {
		// Create the backlog without any member information.
		g, err := state.Guild(ch.guildID)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to get guild")
		}

		// Listen to new members before creating the backlog and requesting members.
		addcancel(ch.session.AddHandler(func(c *gateway.GuildMembersChunkEvent) {
			m, err := ch.session.Store.Messages(ch.id)
			if err != nil {
				// TODO: log
				return
			}

			g, err := ch.session.Store.Guild(c.GuildID)
			if err != nil {
				return
			}

			for _, member := range c.Members {
				// Loop over all messages and replace the author.
				for _, msg := range m {
					if msg.Author.ID != member.User.ID {
						continue
					}

					ct.UpdateMessage(NewMessageUpdateAuthor(msg, member, *g))
				}
			}
		}))

		for _, m := range m {
			ct.CreateMessage(NewBacklogMessage(m, ch.session, *g))
		}

	} else {
		for _, m := range m {
			ct.CreateMessage(NewDirectMessage(m))
		}
	}

	// Bind the handler.
	addcancel(
		ch.session.AddHandler(func(m *gateway.MessageCreateEvent) {
			ct.CreateMessage(NewMessageWithMember(m.Message, ch.session, m.Member))
		}),
		ch.session.AddHandler(func(m *gateway.MessageUpdateEvent) {
			// If the updated content is empty. TODO: add embed support.
			if m.Content == "" {
				return
			}
			ct.UpdateMessage(NewMessageUpdateContent(m.Message))
		}),
		ch.session.AddHandler(func(m *gateway.MessageDeleteEvent) {
			ct.DeleteMessage(NewHeaderDelete(m))
		}),
	)

	return joinCancels(addcancel()), nil
}

func (ch *Channel) SendMessage(msg cchat.SendableMessage) error {
	_, err := ch.session.SendText(ch.id, msg.Content())
	return err
}

func newCancels() func(...func()) []func() {
	var cancels []func()
	return func(appended ...func()) []func() {
		cancels = append(cancels, appended...)
		return cancels
	}
}

func joinCancels(cancellers []func()) func() {
	return func() {
		for _, c := range cancellers {
			c()
		}
	}
}
