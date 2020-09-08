package discord

import (
	"context"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/session"
	"github.com/diamondburned/cchat/services"
	"github.com/diamondburned/cchat/text"
	"github.com/pkg/errors"
)

func init() {
	services.RegisterService(&Service{})
}

// ErrInvalidSession is returned if SessionRestore is given a bad session.
var ErrInvalidSession = errors.New("invalid session")

type Service struct{}

var (
	_ cchat.Iconer  = (*Service)(nil)
	_ cchat.Service = (*Service)(nil)
)

func (Service) Name() text.Rich {
	return text.Rich{Content: "Discord"}
}

// IsIconer returns true.
func (Service) IsIconer() bool { return true }

func (Service) Icon(ctx context.Context, iconer cchat.IconContainer) (func(), error) {
	iconer.SetIcon("https://raw.githubusercontent.com/" +
		"diamondburned/cchat-discord/himearikawa/discord_logo.png")
	return func() {}, nil
}

func (Service) Authenticate() cchat.Authenticator {
	return &Authenticator{}
}

func (s Service) RestoreSession(data map[string]string) (cchat.Session, error) {
	tk, ok := data["token"]
	if !ok {
		return nil, ErrInvalidSession
	}

	return session.NewFromToken(tk)
}

type Authenticator struct{}

var _ cchat.Authenticator = (*Authenticator)(nil)

func (*Authenticator) AuthenticateForm() []cchat.AuthenticateEntry {
	// TODO: username, password and 2FA
	return []cchat.AuthenticateEntry{
		{
			Name:   "Token",
			Secret: true,
		},
		{
			Name: "(or) Username",
		},
	}
}

func (*Authenticator) Authenticate(form []string) (cchat.Session, error) {
	switch {
	case form[0] != "": // Token
		return session.NewFromToken(form[0])
	case form[1] != "": // Username
		return nil, errors.New("username sign-in is not supported yet")
	}

	return nil, errors.New("malformed authentication form")
}
