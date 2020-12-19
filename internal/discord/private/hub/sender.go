package hub

import (
	"regexp"
	"strings"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/channel/message/send"
	"github.com/diamondburned/cchat-discord/internal/discord/channel/message/send/complete"
	"github.com/diamondburned/cchat-discord/internal/discord/state"
	"github.com/diamondburned/cchat-discord/internal/discord/state/nonce"
	"github.com/diamondburned/cchat/utils/empty"
	"github.com/pkg/errors"
)

// ChannelAdder is used to add a new direct message channel into a container.
type ChannelAdder interface {
	AddChannel(state *state.Instance, ch *discord.Channel)
}

// TODO: unexport Sender

type Sender struct {
	empty.Sender
	adder    ChannelAdder
	acList   *activeList
	sentMsgs *nonce.Set
	state    *state.Instance

	completers complete.Completer
}

// mentionRegex matche the following:
//
//    <#123123>
//    <#!12312> // This is OK because we're not sending it.
//    <@123123>
//    <@!12312>
//
var mentionRegex = regexp.MustCompile(`(?m)^<(@|#)!?(\d+)> ?`)

// wrappedMessage wraps around a SendableMessage to override its content.
type wrappedMessage struct {
	cchat.SendableMessage
	content string
}

func (wrMsg wrappedMessage) Content() string {
	return wrMsg.content
}

func (s *Sender) CanAttach() bool { return true }

func (s *Sender) Send(sendable cchat.SendableMessage) error {
	content := sendable.Content()

	// Validate message.
	matches := mentionRegex.FindStringSubmatch(content)
	if matches == nil {
		return errors.New("message must start with a user or channel mention")
	}

	// TODO: account for channel names

	targetID, err := discord.ParseSnowflake(matches[2])
	if err != nil {
		return errors.Wrap(err, "failed to parse recipient ID")
	}

	var channel *discord.Channel
	switch matches[1] {
	case "@":
		channel, _ = s.state.CreatePrivateChannel(discord.UserID(targetID))
	case "#":
		channel, _ = s.state.Channel(discord.ChannelID(targetID))
	}
	if channel == nil {
		return errors.New("unknown channel")
	}

	s.adder.AddChannel(s.state, channel)
	s.acList.add(channel.ID)

	sendData := send.WrapMessage(s.state, sendable)
	sendData.Content = strings.TrimPrefix(content, matches[0])

	// Store the nonce.
	s.sentMsgs.Store(sendData.Nonce)

	_, err = s.state.SendMessageComplex(channel.ID, sendData)
	return errors.Wrap(err, "failed to send message")
}

func (s *Sender) AsCompleter() cchat.Completer {
	return s.completers
}
