// Package state provides a shared state instance for other packages to use.
package state

import (
	"context"
	"log"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/state"
	"github.com/diamondburned/arikawa/utils/httputil/httpdriver"
	"github.com/diamondburned/ningen"
	"github.com/pkg/errors"
)

type Instance struct {
	*ningen.State
	UserID discord.UserID
}

func NewFromToken(token string) (*Instance, error) {
	s, err := state.New(token)
	if err != nil {
		return nil, err
	}

	return New(s)
}

func New(s *state.State) (*Instance, error) {
	// Prefetch user.
	u, err := s.Me()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get current user")
	}

	n, err := ningen.FromState(s)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create a state wrapper")
	}

	n.Client.OnRequest = append(n.Client.OnRequest, func(r httpdriver.Request) error {
		log.Println("[Discord] Request", r.GetPath())
		return nil
	})

	if err := n.Open(); err != nil {
		return nil, err
	}

	return &Instance{
		UserID: u.ID,
		State:  n,
	}, nil
}

// StateOnly returns a shallow copy of *State with an already-expired context.
func (s *Instance) StateOnly() *state.State {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	return s.WithContext(ctx)
}
