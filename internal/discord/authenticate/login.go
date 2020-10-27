package authenticate

import (
	"github.com/diamondburned/arikawa/api"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/session"
	"github.com/diamondburned/cchat-discord/internal/discord/state"
	"github.com/diamondburned/cchat/text"
	"github.com/pkg/errors"
)

// LoginAuthenticator is a first stage authenticator that allows the user to
// authenticate using their email and password.
type LoginAuthenticator struct {
	client *api.Client
}

func NewLoginAuthenticator() *LoginAuthenticator {
	return &LoginAuthenticator{
		client: api.NewClient(""),
	}
}

func (a *LoginAuthenticator) Name() text.Rich {
	return text.Plain("Email")
}

func (a *LoginAuthenticator) Description() text.Rich {
	return text.Plain("Log in using your email.")
}

func (a *LoginAuthenticator) AuthenticateForm() []cchat.AuthenticateEntry {
	return []cchat.AuthenticateEntry{
		{Name: "Email"},
		{Name: "Password", Secret: true},
	}
}

func (a *LoginAuthenticator) Authenticate(form []string) (cchat.Session, cchat.AuthenticateError) {
	if len(form) != 2 {
		return nil, cchat.WrapAuthenticateError(ErrMalformed)
	}

	// Try to login without TOTP
	l, err := a.client.Login(form[0], form[1])
	if err != nil {
		return nil, cchat.WrapAuthenticateError(errors.Wrap(err, "failed to login"))
	}

	if l.MFA {
		return nil, &ErrNeeds2FA{loginResp: l}
	}

	i, err := state.NewFromToken(l.Token)
	if err != nil {
		return nil, cchat.WrapAuthenticateError(errors.Wrap(err, "failed to use token"))
	}

	s, err := session.NewFromInstance(i)
	if err != nil {
		return nil, cchat.WrapAuthenticateError(errors.Wrap(err, "failed to make a session"))
	}

	return s, nil
}
