package edit

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/channel/shared"
	"github.com/pkg/errors"
)

type Editor struct {
	*shared.Channel
}

func New(ch *shared.Channel) cchat.Editor {
	return Editor{ch}
}

// IsEditable returns true if the given message ID belongs to the current
// user.
func (ed Editor) IsEditable(id string) bool {
	s, err := discord.ParseSnowflake(id)
	if err != nil {
		return false
	}

	m, err := ed.State.Store.Message(ed.ID, discord.MessageID(s))
	if err != nil {
		return false
	}

	return m.Author.ID == ed.State.UserID
}

// RawContent returns the raw message content from Discord.
func (ed Editor) RawContent(id string) (string, error) {
	s, err := discord.ParseSnowflake(id)
	if err != nil {
		return "", errors.Wrap(err, "Failed to parse ID")
	}

	m, err := ed.State.Store.Message(ed.ID, discord.MessageID(s))
	if err != nil {
		return "", errors.Wrap(err, "Failed to get the message")
	}

	return m.Content, nil
}

// Edit edits the message to the given content string.
func (ed Editor) Edit(id, content string) error {
	s, err := discord.ParseSnowflake(id)
	if err != nil {
		return errors.Wrap(err, "Failed to parse ID")
	}

	_, err = ed.State.EditText(ed.ID, discord.MessageID(s), content)
	return err
}
