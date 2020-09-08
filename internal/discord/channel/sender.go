package channel

import (
	"github.com/diamondburned/arikawa/api"
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/cchat"
)

var (
	_ cchat.MessageSender    = (*Channel)(nil)
	_ cchat.AttachmentSender = (*Channel)(nil)
)

func (ch *Channel) IsMessageSender() bool {
	p, err := ch.state.StateOnly().Permissions(ch.id, ch.state.UserID)
	if err != nil {
		return false
	}

	return p.Has(discord.PermissionSendMessages)
}

func (ch *Channel) SendMessage(msg cchat.SendableMessage) error {
	var send = api.SendMessageData{Content: msg.Content()}
	if noncer, ok := msg.(cchat.MessageNonce); ok {
		send.Nonce = noncer.Nonce()
	}
	if attcher, ok := msg.(cchat.Attachments); ok {
		send.Files = addAttachments(attcher.Attachments())
	}

	_, err := ch.state.SendMessageComplex(ch.id, send)
	return err
}

// IsAttachmentSender returns true if the channel can attach files.
func (ch *Channel) IsAttachmentSender() bool {
	p, err := ch.state.StateOnly().Permissions(ch.id, ch.state.UserID)
	if err != nil {
		return false
	}

	return p.Has(discord.PermissionAttachFiles)
}

func (ch *Channel) SendAttachments(atts []cchat.MessageAttachment) error {
	_, err := ch.state.SendMessageComplex(ch.id, api.SendMessageData{
		Files: addAttachments(atts),
	})
	return err
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
