package action

import (
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/channel/shared"
	"github.com/pkg/errors"
)

type Actioner struct {
	shared.Channel
}

var _ cchat.Actioner = (*Actioner)(nil)

func New(ch shared.Channel) Actioner {
	return Actioner{ch}
}

const (
	ActionDelete = "Delete"
)

var ErrUnknownAction = errors.New("unknown message action")

func (ac Actioner) Do(action, id string) error {
	s, err := discord.ParseSnowflake(id)
	if err != nil {
		return errors.Wrap(err, "Failed to parse ID")
	}

	switch action {
	case ActionDelete:
		return ac.State.DeleteMessage(ac.ID, discord.MessageID(s))
	default:
		return ErrUnknownAction
	}
}

func (ac Actioner) Actions(id string) []string {
	s, err := discord.ParseSnowflake(id)
	if err != nil {
		return nil
	}

	m, err := ac.State.Cabinet.Message(ac.ID, discord.MessageID(s))
	if err != nil {
		return nil
	}

	// Get the current user.
	u, err := ac.State.Cabinet.Me()
	if err != nil {
		return nil
	}

	// Can we have delete? We can if this is our own message.
	var canDelete = m.Author.ID == u.ID

	// We also can if we have the Manage Messages permission, which would allow
	// us to delete others' messages.
	if !canDelete {
		canDelete = ac.canManageMessages(u.ID)
	}

	if canDelete {
		return []string{ActionDelete}
	}

	return []string{}
}

// canManageMessages returns whether or not the user is allowed to manage
// messages.
func (ac Actioner) canManageMessages(userID discord.UserID) bool {
	// If we're not in a guild, then clearly we cannot.
	if !ac.GuildID.IsValid() {
		return false
	}

	// We need the guild, member and channel to calculate the permission
	// overrides.

	g, err := ac.Guild()
	if err != nil {
		return false
	}

	c, err := ac.Self()
	if err != nil {
		return false
	}

	m, err := ac.State.Cabinet.Member(ac.GuildID, userID)
	if err != nil {
		return false
	}

	p := discord.CalcOverwrites(*g, *c, *m)
	// The Manage Messages permission allows the user to delete others'
	// messages, so we'll return true if that is the case.
	return p.Has(discord.PermissionManageMessages)
}
