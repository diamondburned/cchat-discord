package discord

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/authenticate"
	"github.com/diamondburned/cchat-discord/internal/discord/config"
	"github.com/diamondburned/cchat-discord/internal/discord/session"
	"github.com/diamondburned/cchat-discord/internal/segments/avatar"
	"github.com/diamondburned/cchat/services"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/cchat/utils/empty"
)

var service cchat.Service = Service{}

func init() {
	services.RegisterService(service)
}

// Logo implements cchat.Iconer for the Discord logo.
var Logo = avatar.Segment{
	URL: "https://raw.githubusercontent.com/" +
		"diamondburned/cchat-discord/himearikawa/discord_logo.png",
	Size: 169,
	Text: "Discord",
}

type Service struct {
	empty.Service
}

func (Service) Name() text.Rich {
	return text.Rich{
		Content:  "Discord",
		Segments: []text.Segment{Logo},
	}
}

func (Service) Authenticate() []cchat.Authenticator {
	return authenticate.FirstStageAuthenticators()
}

func (Service) AsSessionRestorer() cchat.SessionRestorer {
	return session.Restorer
}

func (Service) AsConfigurator() cchat.Configurator {
	return config.World
}
