package session

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/state"
)

var Restorer cchat.SessionRestorer = restorer{}

type restorer struct{}

func (restorer) RestoreSession(data map[string]string) (cchat.Session, error) {
	i, err := state.NewFromData(data)
	if err != nil {
		return nil, err
	}

	return NewFromInstance(i)
}
