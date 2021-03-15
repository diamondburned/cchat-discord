package messenger

import (
	"context"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/session/channel/shared"
)

type nicknamer struct {
	userID discord.UserID
	shared.Channel
}

// New creates a new nicknamer for self.
func NewMeNicknamer(ch shared.Channel) cchat.Nicknamer {
	return NewUserNicknamer(ch.State.UserID, ch)
}

// NewUserNicknamer creates a new nicknamer for the given user ID.
func NewUserNicknamer(userID discord.UserID, ch shared.Channel) cchat.Nicknamer {
	return nicknamer{userID, ch}
}

func (nn nicknamer) Nickname(ctx context.Context, labeler cchat.LabelContainer) (func(), error) {
	return nn.State.Labels.AddMemberLabel(nn.GuildID, nn.userID, labeler), nil
}
