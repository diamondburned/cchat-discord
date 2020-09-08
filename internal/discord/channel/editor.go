package channel

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/cchat"
	"github.com/pkg/errors"
)

var _ cchat.Editor = (*Channel)(nil)

// IsEditor returns true if the user can send messages in this channel.
func (ch *Channel) IsEditor() bool {
	p, err := ch.state.StateOnly().Permissions(ch.id, ch.state.UserID)
	if err != nil {
		return false
	}

	return p.Has(discord.PermissionSendMessages)
}

// MessageEditable returns true if the given message ID belongs to the current
// user.
func (ch *Channel) MessageEditable(id string) bool {
	s, err := discord.ParseSnowflake(id)
	if err != nil {
		return false
	}

	m, err := ch.state.Store.Message(ch.id, discord.MessageID(s))
	if err != nil {
		return false
	}

	return m.Author.ID == ch.state.UserID
}

// RawMessageContent returns the raw message content from Discord.
func (ch *Channel) RawMessageContent(id string) (string, error) {
	s, err := discord.ParseSnowflake(id)
	if err != nil {
		return "", errors.Wrap(err, "Failed to parse ID")
	}

	m, err := ch.state.Store.Message(ch.id, discord.MessageID(s))
	if err != nil {
		return "", errors.Wrap(err, "Failed to get the message")
	}

	return m.Content, nil
}

// EditMessage edits the message to the given content string.
func (ch *Channel) EditMessage(id, content string) error {
	s, err := discord.ParseSnowflake(id)
	if err != nil {
		return errors.Wrap(err, "Failed to parse ID")
	}

	_, err = ch.state.EditText(ch.id, discord.MessageID(s), content)
	return err
}
