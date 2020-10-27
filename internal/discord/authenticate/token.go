package authenticate

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/session"
	"github.com/diamondburned/cchat-discord/internal/discord/state"
	"github.com/diamondburned/cchat/text"
	"github.com/pkg/errors"
)

// TokenAuthenticator is a first stage authenticator that allows the user to
// authenticate directly using a token.
type TokenAuthenticator struct{}

func NewTokenAuthenticator() TokenAuthenticator {
	return TokenAuthenticator{}
}

func (TokenAuthenticator) Name() text.Rich {
	return text.Plain("Token")
}

func (TokenAuthenticator) Description() text.Rich {
	return text.Plain("Log in using a token")
}

func (TokenAuthenticator) AuthenticateForm() []cchat.AuthenticateEntry {
	return []cchat.AuthenticateEntry{
		{Name: "Token", Secret: true},
	}
}

func (TokenAuthenticator) Authenticate(form []string) (cchat.Session, cchat.AuthenticateError) {
	if len(form) != 1 {
		return nil, cchat.WrapAuthenticateError(ErrMalformed)
	}

	i, err := state.NewFromToken(form[0])
	if err != nil {
		return nil, cchat.WrapAuthenticateError(errors.Wrap(err, "failed to use token"))
	}

	s, err := session.NewFromInstance(i)
	if err != nil {
		return nil, cchat.WrapAuthenticateError(errors.Wrap(err, "failed to make a session"))
	}

	return s, nil
}
