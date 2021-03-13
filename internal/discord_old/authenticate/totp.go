package authenticate

import (
	"github.com/diamondburned/arikawa/v2/api"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/session"
	"github.com/diamondburned/cchat-discord/internal/discord/state"
	"github.com/diamondburned/cchat/text"
	"github.com/pkg/errors"
)

// ErrNeeds2FA is returned from Authenticator if the user login requires a 2FA
// token.
type ErrNeeds2FA struct {
	loginResp *api.LoginResponse
}

func (err ErrNeeds2FA) Error() string {
	return "Two-Factor Authentication token required"
}

func (err ErrNeeds2FA) NextStage() []cchat.Authenticator {
	return []cchat.Authenticator{
		NewTOTPAuthenticator(err.loginResp.Ticket),
	}
}

// TOTPAuthenticator is a second stage authenticator that follows the normal
// Authenticator if the user has Two-Factor Authentication enabled.
type TOTPAuthenticator struct {
	client *api.Client
	ticket string
}

func NewTOTPAuthenticator(ticket string) *TOTPAuthenticator {
	return &TOTPAuthenticator{
		client: api.NewClient(""),
		ticket: ticket,
	}
}

func (auth *TOTPAuthenticator) Name() text.Rich {
	return text.Plain("2FA Prompt")
}

func (auth *TOTPAuthenticator) Description() text.Rich {
	return text.Plain("Enter your 2FA token.")
}

func (auth *TOTPAuthenticator) AuthenticateForm() []cchat.AuthenticateEntry {
	return []cchat.AuthenticateEntry{
		{Name: "Token", Description: "6-digit code"},
	}
}

func (auth *TOTPAuthenticator) Authenticate(v []string) (cchat.Session, cchat.AuthenticateError) {
	if len(v) != 1 {
		return nil, cchat.WrapAuthenticateError(ErrMalformed)
	}

	l, err := auth.client.TOTP(v[0], auth.ticket)
	if err != nil {
		return nil, cchat.WrapAuthenticateError(errors.Wrap(err, "failed to login with 2FA"))
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
