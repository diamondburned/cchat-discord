package discord

import (
	"context"

	"github.com/diamondburned/cchat"
)

const LogoURL = "https://raw.githubusercontent.com/" +
	"diamondburned/cchat-discord/himearikawa/discord_logo.png"

// Logo implements cchat.Iconer for the Discord logo.
var Logo cchat.Iconer = logo{}

type logo struct{}

func (logo) Icon(ctx context.Context, iconer cchat.IconContainer) (func(), error) {
	iconer.SetIcon(LogoURL)
	return func() {}, nil
}
