package hub

import (
	"regexp"
	"strings"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/session/channel/messenger/sender"
	"github.com/diamondburned/cchat-discord/internal/discord/session/channel/messenger/sender/completer"
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

	completers completer.Completer
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

	switch channel.Type {
	case discord.DirectMessage, discord.GroupDM:
		// valid
	default:
		return errors.New("not a [group] direct message channel")
	}

	// We should only add the channel if it's not already in the active list.
	if s.acList.add(channel.ID) {
		s.adder.AddChannel(s.state, channel)
	}

	sendData := sender.WrapMessage(s.state, channel.ID, sendable)
	sendData.Content = strings.TrimPrefix(content, matches[0])

	// Store the nonce.
	s.sentMsgs.Store(sendData.Nonce)

	_, err = s.state.SendMessageComplex(channel.ID, sendData)
	return errors.Wrap(err, "failed to send message")
}

func (s *Sender) AsCompleter() cchat.Completer {
	return s.completers
}
