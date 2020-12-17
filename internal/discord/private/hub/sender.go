package hub

import (
	"regexp"
	"strings"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/channel/message/send"
	"github.com/diamondburned/cchat-discord/internal/discord/state"
	"github.com/diamondburned/cchat/utils/empty"
	"github.com/pkg/errors"
)

// ChannelAdder is used to add a new direct message channel into a container.
type ChannelAdder interface {
	AddChannel(state *state.Instance, ch *discord.Channel)
}

type Sender struct {
	empty.Sender
	adder  ChannelAdder
	acList *activeList
	state  *state.Instance
}

func NewSender(s *state.Instance, acList *activeList, adder ChannelAdder) *Sender {
	return &Sender{adder: adder, acList: acList, state: s}
}

var mentionRegex = regexp.MustCompile(`^<@!?(\d+)> ?`)

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
		return errors.New("messages sent here must start with a mention")
	}

	targetID, err := discord.ParseSnowflake(matches[1])
	if err != nil {
		return errors.Wrap(err, "failed to parse recipient ID")
	}

	ch, err := s.state.CreatePrivateChannel(discord.UserID(targetID))
	if err != nil {
		return errors.Wrap(err, "failed to find DM channel")
	}

	s.adder.AddChannel(s.state, ch)
	s.acList.add(ch.ID)

	return send.Send(s.state, ch.ID, wrappedMessage{
		SendableMessage: sendable,
		content:         strings.TrimPrefix(content, matches[0]),
	})
}

// func (msgs *Messages) AsCompleter() cchat.Completer {
// 	return complete.New(msgs)
// }
