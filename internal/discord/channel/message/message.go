package message

import (
	"context"
	"sort"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/channel/message/action"
	"github.com/diamondburned/cchat-discord/internal/discord/channel/message/backlog"
	"github.com/diamondburned/cchat-discord/internal/discord/channel/message/edit"
	"github.com/diamondburned/cchat-discord/internal/discord/channel/message/indicate"
	"github.com/diamondburned/cchat-discord/internal/discord/channel/message/memberlist"
	"github.com/diamondburned/cchat-discord/internal/discord/channel/message/nickname"
	"github.com/diamondburned/cchat-discord/internal/discord/channel/message/send"
	"github.com/diamondburned/cchat-discord/internal/discord/channel/shared"
	"github.com/diamondburned/cchat-discord/internal/discord/message"
	"github.com/diamondburned/cchat-discord/internal/funcutil"
	"github.com/diamondburned/cchat/utils/empty"
	"github.com/pkg/errors"
)

type Messenger struct {
	empty.Messenger
	shared.Channel
}

var _ cchat.Messenger = (*Messenger)(nil)

func New(ch shared.Channel) Messenger {
	return Messenger{Channel: ch}
}

func (msgr Messenger) JoinServer(ctx context.Context, ct cchat.MessagesContainer) (func(), error) {
	state := msgr.State.WithContext(ctx)

	m, err := state.Messages(msgr.ID)
	if err != nil {
		return nil, err
	}

	var addcancel = funcutil.NewCancels()

	var constructor func(discord.Message) cchat.MessageCreate

	if msgr.GuildID.IsValid() {
		// Create the backlog without any member information.
		g, err := state.Guild(msgr.GuildID)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to get guild")
		}

		constructor = func(m discord.Message) cchat.MessageCreate {
			return message.NewBacklogMessage(m, msgr.State, *g)
		}

		// Subscribe to typing events.
		msgr.State.MemberState.Subscribe(msgr.GuildID)

		// Listen to new members before creating the backlog and requesting members.
		addcancel(msgr.State.AddHandler(func(c *gateway.GuildMembersChunkEvent) {
			if c.GuildID != msgr.GuildID {
				return
			}

			m, err := msgr.Messages()
			if err != nil {
				// TODO: log
				return
			}

			g, err := msgr.Guild()
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

					ct.UpdateMessage(message.NewMessageUpdateAuthor(msg, member, *g, msgr.State))
				}
			}
		}))
	} else {
		constructor = func(m discord.Message) cchat.MessageCreate {
			return message.NewDirectMessage(m, msgr.State)
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
		msgr.State.ReadState.MarkRead(msgr.ID, m[len(m)-1].ID)
	}

	// Bind the handler.
	addcancel(
		msgr.State.AddHandler(func(m *gateway.MessageCreateEvent) {
			if m.ChannelID == msgr.ID {
				ct.CreateMessage(message.NewMessageCreate(m, msgr.State))
				msgr.State.ReadState.MarkRead(msgr.ID, m.ID)
			}
		}),
		msgr.State.AddHandler(func(m *gateway.MessageUpdateEvent) {
			// If the updated content is empty. TODO: add embed support.
			if m.ChannelID == msgr.ID {
				ct.UpdateMessage(message.NewMessageUpdateContent(m.Message, msgr.State))
			}
		}),
		msgr.State.AddHandler(func(m *gateway.MessageDeleteEvent) {
			if m.ChannelID == msgr.ID {
				ct.DeleteMessage(message.NewHeaderDelete(m))
			}
		}),
	)

	return funcutil.JoinCancels(addcancel()...), nil
}

func (msgr Messenger) AsSender() cchat.Sender {
	if !msgr.HasPermission(discord.PermissionSendMessages) {
		return nil
	}

	return send.New(msgr.Channel)
}

func (msgr Messenger) AsEditor() cchat.Editor {
	if !msgr.HasPermission(discord.PermissionSendMessages) {
		return nil
	}

	return edit.New(msgr.Channel)
}

func (msgr Messenger) AsActioner() cchat.Actioner {
	return action.New(msgr.Channel)
}

func (msgr Messenger) AsNicknamer() cchat.Nicknamer {
	return nickname.New(msgr.Channel)
}

func (msgr Messenger) AsMemberLister() cchat.MemberLister {
	if !msgr.GuildID.IsValid() {
		return nil
	}
	return memberlist.New(msgr.Channel)
}

func (msgr Messenger) AsBacklogger() cchat.Backlogger {
	if !msgr.HasPermission(discord.PermissionViewChannel, discord.PermissionReadMessageHistory) {
		return nil
	}

	return backlog.New(msgr.Channel)
}

func (msgr Messenger) AsTypingIndicator() cchat.TypingIndicator {
	return indicate.NewTyping(msgr.Channel)
}

func (msgr Messenger) AsUnreadIndicator() cchat.UnreadIndicator {
	return indicate.NewUnread(msgr.Channel)
}
