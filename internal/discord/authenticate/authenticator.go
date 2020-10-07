package authenticate

import (
	"errors"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/session"
	"github.com/diamondburned/cchat-discord/internal/discord/state"
)

var (
	ErrMalformed  = errors.New("malformed authentication form")
	EnterPassword = errors.New("enter your password")
)

type Authenticator struct {
	username string
	password string
}

func New() cchat.Authenticator {
	return &Authenticator{}
}

func (a *Authenticator) stage() int {
	switch {
	// Stage 1: Prompt for the token OR username.
	case a.username == "" && a.password == "":
		return 0

	// Stage 2: Prompt for the password.
	case a.password == "":
		return 1

	// Stage 3: Prompt for the TOTP token.
	default:
		return 2
	}
}

func (a *Authenticator) AuthenticateForm() []cchat.AuthenticateEntry {
	switch a.stage() {
	case 0:
		return []cchat.AuthenticateEntry{
			{Name: "Token", Secret: true},
			{Name: "Username", Description: "Fill either Token or Username only."},
		}
	case 1:
		return []cchat.AuthenticateEntry{
			{Name: "Password", Secret: true},
		}
	case 2:
		return []cchat.AuthenticateEntry{
			{Name: "Auth Code", Description: "6-digit code for Two-factor Authentication."},
		}
	default:
		return nil
	}
}

func (a *Authenticator) Authenticate(form []string) (cchat.Session, error) {
	switch a.stage() {
	case 0:
		if len(form) != 2 {
			return nil, ErrMalformed
		}

		switch {
		case form[0] != "": // Token
			i, err := state.NewFromToken(form[0])
			if err != nil {
				return nil, err
			}

			return session.NewFromInstance(i)

		case form[1] != "": // Username
			// Move to a new stage.
			a.username = form[1]
			return nil, EnterPassword
		}

	case 1:
		if len(form) != 1 {
			return nil, ErrMalformed
		}

		a.password = form[0]

		i, err := state.Login(a.username, a.password, "")
		if err != nil {
			// If the error is not ErrMFA, then we should reset password to
			// empty.
			if !errors.Is(err, session.ErrMFA) {
				a.password = ""
			}

			return nil, err
		}

		return session.NewFromInstance(i)

	case 2:
		if len(form) != 1 {
			return nil, ErrMalformed
		}

		i, err := state.Login(a.username, a.password, form[0])
		if err != nil {
			return nil, err
		}

		return session.NewFromInstance(i)
	}

	return nil, ErrMalformed
}
