// Package state provides a shared state instance for other packages to use.
package state

import (
	"context"
	"log"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/session"
	"github.com/diamondburned/arikawa/v2/state"
	"github.com/diamondburned/arikawa/v2/state/store/defaultstore"
	"github.com/diamondburned/arikawa/v2/utils/httputil/httpdriver"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/state/labels"
	"github.com/diamondburned/cchat-discord/internal/discord/state/nonce"
	"github.com/diamondburned/ningen/v2"
	"github.com/pkg/errors"
)

type Instance struct {
	*ningen.State
	Nonces *nonce.Map
	Labels *labels.Repository

	// UserID is a constant user ID of the current user. It is guaranteed to be
	// valid.
	UserID discord.UserID
}

var _ cchat.SessionSaver = (*Instance)(nil)

// ErrInvalidSession is returned if SessionRestore is given a bad session.
var ErrInvalidSession = errors.New("invalid session")

func NewFromData(data map[string]string) (*Instance, error) {
	tk, ok := data["token"]
	if !ok {
		return nil, ErrInvalidSession
	}

	return NewFromToken(tk)
}

func NewFromToken(token string) (*Instance, error) {
	s, err := state.New(token)
	if err != nil {
		return nil, err
	}

	return New(s)
}

func Login(email, password, mfa string) (*Instance, error) {
	session, err := session.Login(email, password, mfa)
	if err != nil {
		return nil, err
	}

	cabinet := defaultstore.New()
	cabinet.MessageStore = defaultstore.NewMessage(50)

	return New(state.NewFromSession(session, cabinet))
}

func New(s *state.State) (*Instance, error) {
	n, err := ningen.FromState(s)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create a state wrapper")
	}

	n.Client.OnRequest = append(n.Client.OnRequest,
		func(r httpdriver.Request) error {
			log.Println("[Discord] Request", r.GetPath())
			return nil
		},
	)

	if err := n.Open(); err != nil {
		return nil, err
	}

	// Prefetch user.
	u, err := s.Me()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current user")
	}

	return &Instance{
		UserID: u.ID,
		State:  n,
		Nonces: new(nonce.Map),
		Labels: labels.NewRepository(n),
	}, nil
}

// Permissions queries for the permission without hitting the REST API.
func (s *Instance) Permissions(
	chID discord.ChannelID, uID discord.UserID) (discord.Permissions, error) {

	return s.StateOnly().Permissions(chID, uID)
}

var deadCtx = expiredContext()

// StateOnly returns a shallow copy of *State with an already-expired context.
func (s *Instance) StateOnly() *state.State {
	return s.WithContext(deadCtx)
}

func (s *Instance) SaveSession() map[string]string {
	return map[string]string{
		"token": s.Token,
	}
}

func expiredContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	return ctx
}
