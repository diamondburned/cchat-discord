package authenticate

import (
	"errors"

	"github.com/diamondburned/cchat"
)

var (
	ErrMalformed  = errors.New("malformed authentication form")
	EnterPassword = errors.New("enter your password")
)

// FirstStageAuthenticators constructs a slice of newly made first stage
// authenticators.
func FirstStageAuthenticators() []cchat.Authenticator {
	return []cchat.Authenticator{
		NewTokenAuthenticator(),
		NewLoginAuthenticator(),
		NewDiscordLogin(),
	}
}
