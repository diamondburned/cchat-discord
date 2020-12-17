package hub

import (
	"context"
	"sync"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/message"
	"github.com/diamondburned/cchat-discord/internal/discord/state"
	"github.com/diamondburned/cchat-discord/internal/funcutil"
	"github.com/diamondburned/cchat/utils/empty"
)

const maxMessages = 100

type messageList []discord.Message

func (list messageList) idx(id discord.MessageID) int {
	for i, msg := range list {
		if msg.ID == id {
			return i
		}
	}
	return -1
}

func (list *messageList) append(msg discord.Message) {
	*list = append(*list, msg)

	// cap the length
	if len(*list) > maxMessages {
		copy(*list, (*list)[1:])                  // shift left once
		(*list)[len(*list)-1] = discord.Message{} // nil out last to not memory leak
		*list = (*list)[:len(*list)-1]            // slice it away
	}
}

func (list *messageList) swap(newMsg discord.Message) {
	if idx := list.idx(newMsg.ID); idx > -1 {
		(*list)[idx] = newMsg
	}
}

func (list *messageList) delete(id discord.MessageID) {
	if idx := list.idx(id); idx > -1 {
		*list = append((*list)[:idx], (*list)[idx+1:]...)
	}
}

type Messages struct {
	empty.Messenger

	state  *state.Instance
	acList *activeList

	sender *Sender

	msgMutex sync.Mutex
	messages messageList

	cancel func()
}

func NewMessages(s *state.Instance, acList *activeList, adder ChannelAdder) *Messages {
	hubServer := &Messages{
		state:    s,
		acList:   acList,
		sender:   NewSender(s, acList, adder),
		messages: make(messageList, 0, 100),
	}

	hubServer.cancel = funcutil.JoinCancels(
		s.AddHandler(func(msg *gateway.MessageCreateEvent) {
			if msg.GuildID.IsValid() || acList.isActive(msg.ChannelID) {
				return
			}

			hubServer.msgMutex.Lock()
			hubServer.messages.append(msg.Message)
			hubServer.msgMutex.Unlock()
		}),
		s.AddHandler(func(update *gateway.MessageUpdateEvent) {
			if update.GuildID.IsValid() || acList.isActive(update.ChannelID) {
				return
			}

			// The event itself is unreliable, so we must rely on the state.
			m, err := hubServer.state.Message(update.ChannelID, update.ID)
			if err != nil {
				return
			}

			hubServer.msgMutex.Lock()
			hubServer.messages.swap(*m)
			hubServer.msgMutex.Unlock()
		}),
		s.AddHandler(func(del *gateway.MessageDeleteEvent) {
			if del.GuildID.IsValid() || acList.isActive(del.ChannelID) {
				return
			}

			hubServer.msgMutex.Lock()
			hubServer.messages.delete(del.ID)
			hubServer.msgMutex.Unlock()
		}),
	)

	return hubServer
}

func (msgs *Messages) JoinServer(ctx context.Context, ct cchat.MessagesContainer) (func(), error) {
	msgs.msgMutex.Lock()

	for _, msg := range msgs.messages {
		ct.CreateMessage(message.NewDirectMessage(msg, msgs.state))
	}

	msgs.msgMutex.Unlock()

	// Bind the handler.
	return funcutil.JoinCancels(
		msgs.state.AddHandler(func(msg *gateway.MessageCreateEvent) {
			if msg.GuildID.IsValid() || msgs.acList.isActive(msg.ChannelID) {
				return
			}

			ct.CreateMessage(message.NewMessageCreate(msg, msgs.state))
			msgs.state.ReadState.MarkRead(msg.ChannelID, msg.ID)
		}),
		msgs.state.AddHandler(func(update *gateway.MessageUpdateEvent) {
			if update.GuildID.IsValid() || msgs.acList.isActive(update.ChannelID) {
				return
			}

			ct.UpdateMessage(message.NewMessageUpdateContent(update.Message, msgs.state))
		}),
		msgs.state.AddHandler(func(del *gateway.MessageDeleteEvent) {
			if del.GuildID.IsValid() || msgs.acList.isActive(del.ChannelID) {
				return
			}

			ct.DeleteMessage(message.NewHeaderDelete(del))
		}),
	), nil
}

func (msgs *Messages) AsSender() cchat.Sender { return msgs.sender }
