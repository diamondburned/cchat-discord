package messenger

import (
	"context"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/message"
	"github.com/diamondburned/cchat-discord/internal/discord/session/channel/messenger/actioner"
	"github.com/diamondburned/cchat-discord/internal/discord/session/channel/messenger/backlogger"
	"github.com/diamondburned/cchat-discord/internal/discord/session/channel/messenger/editor"
	"github.com/diamondburned/cchat-discord/internal/discord/session/channel/messenger/indicator"
	"github.com/diamondburned/cchat-discord/internal/discord/session/channel/messenger/memberlister"
	"github.com/diamondburned/cchat-discord/internal/discord/session/channel/messenger/sender"
	"github.com/diamondburned/cchat-discord/internal/discord/session/channel/shared"
	"github.com/diamondburned/cchat-discord/internal/funcutil"
	"github.com/diamondburned/cchat/utils/empty"
)

type Messenger struct {
	empty.Messenger
	shared.Channel
}

var _ cchat.Messenger = (*Messenger)(nil)

func New(ch shared.Channel) *Messenger {
	return &Messenger{Channel: ch}
}

func (msgr *Messenger) JoinServer(ctx context.Context, ct cchat.MessagesContainer) (func(), error) {
	state := msgr.State.WithContext(ctx)

	m, err := state.Messages(msgr.ID)
	if err != nil {
		return nil, err
	}

	addcancel := funcutil.NewCancels()

	if msgr.GuildID.IsValid() {
		// Subscribe to typing events.
		msgr.State.MemberState.Subscribe(msgr.GuildID)

		// Listen to new members before creating the backlog and requesting members.
		addcancel(msgr.State.AddHandler(func(c *gateway.GuildMembersChunkEvent) {
			if c.GuildID != msgr.GuildID {
				return
			}

			messages, _ := msgr.Messages()

			for _, m := range c.Members {
				for _, msg := range messages {
					if msg.Author.ID == m.User.ID {
						ct.UpdateMessage(message.NewAuthorUpdate(msg, m, msgr.State))
					}
				}
			}
		}))
	}

	// Iterate from the earliest messages to the latest messages.
	for _, m := range m {
		ct.CreateMessage(message.NewBacklogMessage(m, msgr.State))
	}

	// Bind the handler.
	addcancel(
		msgr.State.AddHandler(func(m *gateway.MessageCreateEvent) {
			if m.ChannelID == msgr.ID {
				ct.CreateMessage(message.NewGuildMessageCreate(m, msgr.State))
			}
		}),
		msgr.State.AddHandler(func(m *gateway.MessageUpdateEvent) {
			// If the updated content is empty. TODO: add embed support.
			if m.ChannelID == msgr.ID {
				ct.UpdateMessage(message.NewContentUpdate(m.Message, msgr.State))
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

func (msgr *Messenger) AsSender() cchat.Sender {
	if !msgr.HasPermission(discord.PermissionSendMessages) {
		return nil
	}

	return sender.New(msgr.Channel)
}

func (msgr *Messenger) AsEditor() cchat.Editor {
	if !msgr.HasPermission(discord.PermissionSendMessages) {
		return nil
	}

	return editor.New(msgr.Channel)
}

func (msgr *Messenger) AsActioner() cchat.Actioner {
	return actioner.New(msgr.Channel)
}

func (msgr *Messenger) AsNicknamer() cchat.Nicknamer {
	return NewMeNicknamer(msgr.Channel)
}

func (msgr *Messenger) AsMemberLister() cchat.MemberLister {
	if !msgr.GuildID.IsValid() {
		return nil
	}
	return memberlister.New(msgr.Channel)
}

func (msgr *Messenger) AsBacklogger() cchat.Backlogger {
	if !msgr.HasPermission(discord.PermissionViewChannel, discord.PermissionReadMessageHistory) {
		return nil
	}

	return backlogger.New(msgr.Channel)
}

func (msgr *Messenger) AsTypingIndicator() cchat.TypingIndicator {
	return indicator.NewTyping(msgr.Channel)
}

func (msgr *Messenger) AsUnreadIndicator() cchat.UnreadIndicator {
	return indicator.NewUnread(msgr.Channel)
}
