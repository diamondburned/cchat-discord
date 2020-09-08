package channel

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/cchat"
	"github.com/pkg/errors"
)

var _ cchat.Actioner = (*Channel)(nil)

// IsActioner returns true.
func (ch *Channel) IsActioner() bool { return true }

const (
	ActionDelete = "Delete"
)

var ErrUnknownAction = errors.New("unknown message action")

func (ch *Channel) DoMessageAction(action, id string) error {
	s, err := discord.ParseSnowflake(id)
	if err != nil {
		return errors.Wrap(err, "Failed to parse ID")
	}

	switch action {
	case ActionDelete:
		return ch.state.DeleteMessage(ch.id, discord.MessageID(s))
	default:
		return ErrUnknownAction
	}
}

func (ch *Channel) MessageActions(id string) []string {
	s, err := discord.ParseSnowflake(id)
	if err != nil {
		return nil
	}

	m, err := ch.state.Store.Message(ch.id, discord.MessageID(s))
	if err != nil {
		return nil
	}

	// Get the current user.
	u, err := ch.state.Store.Me()
	if err != nil {
		return nil
	}

	// Can we have delete? We can if this is our own message.
	var canDelete = m.Author.ID == u.ID

	// We also can if we have the Manage Messages permission, which would allow
	// us to delete others' messages.
	if !canDelete {
		canDelete = ch.canManageMessages(u.ID)
	}

	if canDelete {
		return []string{ActionDelete}
	}

	return []string{}
}

// canManageMessages returns whether or not the user is allowed to manage
// messages.
func (ch *Channel) canManageMessages(userID discord.UserID) bool {
	// If we're not in a guild, then clearly we cannot.
	if !ch.guildID.IsValid() {
		return false
	}

	// We need the guild, member and channel to calculate the permission
	// overrides.

	g, err := ch.guild()
	if err != nil {
		return false
	}

	c, err := ch.self()
	if err != nil {
		return false
	}

	m, err := ch.state.Store.Member(ch.guildID, userID)
	if err != nil {
		return false
	}

	p := discord.CalcOverwrites(*g, *c, *m)
	// The Manage Messages permission allows the user to delete others'
	// messages, so we'll return true if that is the case.
	return p.Has(discord.PermissionManageMessages)
}
