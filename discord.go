package discord

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/authenticate"
	"github.com/diamondburned/cchat-discord/internal/discord/session"
	"github.com/diamondburned/cchat/services"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/cchat/utils/empty"
)

var service cchat.Service = Service{}

func init() {
	services.RegisterService(service)
}

type Service struct {
	empty.Service
}

func (Service) Name() text.Rich {
	return text.Rich{Content: "Discord"}
}

func (Service) Authenticate() []cchat.Authenticator {
	return authenticate.FirstStageAuthenticators()
}

func (Service) AsIconer() cchat.Iconer {
	return Logo
}

func (Service) AsSessionRestorer() cchat.SessionRestorer {
	return session.Restorer
}
