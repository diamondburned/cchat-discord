package send

import (
	"github.com/diamondburned/arikawa/v2/api"
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/utils/sendpart"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/channel/message/send/complete"
	"github.com/diamondburned/cchat-discord/internal/discord/channel/shared"
	"github.com/diamondburned/cchat-discord/internal/discord/state"
)

func WrapMessage(s *state.Instance, msg cchat.SendableMessage) api.SendMessageData {
	var send = api.SendMessageData{Content: msg.Content()}

	if attacher := msg.AsAttacher(); attacher != nil {
		send.Files = addAttachments(attacher.Attachments())
	}

	if noncer := msg.AsNoncer(); noncer != nil {
		send.Nonce = s.Nonces.Generate(noncer.Nonce())
	}

	if replier := msg.AsReplier(); replier != nil {
		id, err := discord.ParseSnowflake(replier.ReplyingTo())
		if err == nil {
			send.Reference = &discord.MessageReference{
				MessageID: discord.MessageID(id),
			}
		}
	}

	return send
}

type Sender struct {
	shared.Channel
}

var _ cchat.Sender = (*Sender)(nil)

func New(ch shared.Channel) Sender {
	return Sender{ch}
}

func (s Sender) Send(msg cchat.SendableMessage) error {
	_, err := s.State.SendMessageComplex(s.ID, WrapMessage(s.State, msg))
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

func addAttachments(atts []cchat.MessageAttachment) []sendpart.File {
	var files = make([]sendpart.File, len(atts))
	for i, a := range atts {
		files[i] = sendpart.File{
			Name:   a.Name,
			Reader: a,
		}
	}
	return files
}
