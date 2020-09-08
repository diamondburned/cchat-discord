package channel

import (
	"context"
	"sort"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/message"
	"github.com/diamondburned/cchat-discord/internal/funcutil"
	"github.com/pkg/errors"
)

var _ cchat.Messenger = (*Channel)(nil)

// IsMessenger returns true if the current user is allowed to see the channel.
func (ch *Channel) IsMessenger() bool {
	p, err := ch.state.StateOnly().Permissions(ch.id, ch.state.UserID)
	if err != nil {
		return false
	}

	return p.Has(discord.PermissionViewChannel)
}

func (ch *Channel) JoinServer(ctx context.Context, ct cchat.MessagesContainer) (func(), error) {
	state := ch.state.WithContext(ctx)

	m, err := state.Messages(ch.id)
	if err != nil {
		return nil, err
	}

	var addcancel = funcutil.NewCancels()

	var constructor func(discord.Message) cchat.MessageCreate

	if ch.guildID.IsValid() {
		// Create the backlog without any member information.
		g, err := state.Guild(ch.guildID)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to get guild")
		}

		constructor = func(m discord.Message) cchat.MessageCreate {
			return message.NewBacklogMessage(m, ch.state, *g)
		}

		// Subscribe to typing events.
		ch.state.MemberState.Subscribe(ch.guildID)

		// Listen to new members before creating the backlog and requesting members.
		addcancel(ch.state.AddHandler(func(c *gateway.GuildMembersChunkEvent) {
			if c.GuildID != ch.guildID {
				return
			}

			m, err := ch.messages()
			if err != nil {
				// TODO: log
				return
			}

			g, err := ch.guild()
			if err != nil {
				return
			}

			// Loop over all messages and replace the author. The latest
			// messages are in front.
			for _, msg := range m {
				for _, member := range c.Members {
					if msg.Author.ID != member.User.ID {
						continue
					}

					ct.UpdateMessage(message.NewMessageUpdateAuthor(msg, member, *g, ch.state))
				}
			}
		}))
	} else {
		constructor = func(m discord.Message) cchat.MessageCreate {
			return message.NewDirectMessage(m, ch.state)
		}
	}

	// Only do all this if we even have any messages.
	if len(m) > 0 {
		// Sort messages chronologically using the ID so that the oldest messages
		// (ones with the smallest snowflake) is in front.
		sort.Slice(m, func(i, j int) bool { return m[i].ID < m[j].ID })

		// Iterate from the earliest messages to the latest messages.
		for _, m := range m {
			ct.CreateMessage(constructor(m))
		}

		// Mark this channel as read.
		ch.state.ReadState.MarkRead(ch.id, m[len(m)-1].ID)
	}

	// Bind the handler.
	addcancel(
		ch.state.AddHandler(func(m *gateway.MessageCreateEvent) {
			if m.ChannelID == ch.id {
				ct.CreateMessage(message.NewMessageCreate(m, ch.state))
				ch.state.ReadState.MarkRead(ch.id, m.ID)
			}
		}),
		ch.state.AddHandler(func(m *gateway.MessageUpdateEvent) {
			// If the updated content is empty. TODO: add embed support.
			if m.ChannelID == ch.id {
				ct.UpdateMessage(message.NewMessageUpdateContent(m.Message, ch.state))
			}
		}),
		ch.state.AddHandler(func(m *gateway.MessageDeleteEvent) {
			if m.ChannelID == ch.id {
				ct.DeleteMessage(message.NewHeaderDelete(m))
			}
		}),
	)

	return funcutil.JoinCancels(addcancel()), nil
}
