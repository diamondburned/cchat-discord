package send

import (
	"github.com/diamondburned/arikawa/api"
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/channel/message/send/complete"
	"github.com/diamondburned/cchat-discord/internal/discord/channel/shared"
	"github.com/diamondburned/cchat-discord/internal/discord/state"
)

type Sender struct {
	shared.Channel
}

var _ cchat.Sender = (*Sender)(nil)

func New(ch shared.Channel) Sender {
	return Sender{ch}
}

func (s Sender) Send(msg cchat.SendableMessage) error {
	return Send(s.State, s.ID, msg)
}

func Send(s *state.Instance, chID discord.ChannelID, msg cchat.SendableMessage) error {
	var send = api.SendMessageData{Content: msg.Content()}
	if attacher := msg.AsAttachments(); attacher != nil {
		send.Files = addAttachments(attacher.Attachments())
	}

	_, err := s.SendMessageComplex(chID, send)
	return err
}

// CanAttach returns true if the channel can attach files.
func (s Sender) CanAttach() bool {
	p, err := s.State.StateOnly().Permissions(s.ID, s.State.UserID)
	if err != nil {
		return false
	}

	return p.Has(discord.PermissionAttachFiles)
}

func (s Sender) AsCompleter() cchat.Completer {
	return complete.New(s.Channel)
}

func addAttachments(atts []cchat.MessageAttachment) []api.SendMessageFile {
	var files = make([]api.SendMessageFile, len(atts))
	for i, a := range atts {
		files[i] = api.SendMessageFile{
			Name:   a.Name,
			Reader: a,
		}
	}
	return files
}
