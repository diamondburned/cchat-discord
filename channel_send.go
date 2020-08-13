package discord

import (
	"github.com/diamondburned/arikawa/api"
	"github.com/diamondburned/cchat"
)

type SendableChannel struct {
	Channel
}

// NewSendableChannel creates a sendable channel. This function is mainly used
// internally
func NewSendableChannel(ch *Channel) *SendableChannel {
	return &SendableChannel{*ch}
}

var (
	_ cchat.ServerMessageSender           = (*SendableChannel)(nil)
	_ cchat.ServerMessageSendCompleter    = (*SendableChannel)(nil)
	_ cchat.ServerMessageAttachmentSender = (*SendableChannel)(nil)
)

func (ch *SendableChannel) SendMessage(msg cchat.SendableMessage) error {
	var send = api.SendMessageData{Content: msg.Content()}
	if noncer, ok := msg.(cchat.MessageNonce); ok {
		send.Nonce = noncer.Nonce()
	}
	if attcher, ok := msg.(cchat.SendableMessageAttachments); ok {
		send.Files = addAttachments(attcher.Attachments())
	}

	_, err := ch.session.SendMessageComplex(ch.id, send)
	return err
}

func (ch *SendableChannel) SendAttachments(atts []cchat.MessageAttachment) error {
	_, err := ch.session.SendMessageComplex(ch.id, api.SendMessageData{
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

// CompleteMessage implements message input completion capability for Discord.
// This method supports user mentions, channel mentions and emojis.
//
// For the individual implementations, refer to channel_completion.go.
func (ch *SendableChannel) CompleteMessage(words []string, i int) (entries []cchat.CompletionEntry) {
	var word = words[i]
	// Word should have at least a character for the char check.
	if len(word) < 1 {
		return
	}

	switch word[0] {
	case '@':
		return ch.completeMentions(word[1:])
	case '#':
		return ch.completeChannels(word[1:])
	case ':':
		return ch.completeEmojis(word[1:])
	}

	return
}
