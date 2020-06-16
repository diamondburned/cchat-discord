package discord

import (
	"context"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/state"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/ningen"
	"github.com/pkg/errors"
)

type Service struct{}

var (
	_ cchat.Icon    = (*Service)(nil)
	_ cchat.Service = (*Service)(nil)
)

func (Service) Name() text.Rich {
	return text.Rich{Content: "Discord"}
}

func (Service) Icon(ctx context.Context, iconer cchat.IconContainer) error {
	iconer.SetIcon("https://discord.com/assets/2c21aeda16de354ba5334551a883b481.png")
	return nil
}

func (Service) Authenticate() cchat.Authenticator {
	return Authenticator{}
}

type Authenticator struct{}

var _ cchat.Authenticator = (*Authenticator)(nil)

func (Authenticator) AuthenticateForm() []cchat.AuthenticateEntry {
	// TODO: username, password and 2FA
	return []cchat.AuthenticateEntry{
		{
			Name:   "Token",
			Secret: true,
		},
	}
}

func (Authenticator) Authenticate(form []string) (cchat.Session, error) {
	s, err := state.New(form[0])
	if err != nil {
		return nil, err
	}

	if err := s.Open(); err != nil {
		return nil, err
	}

	return NewSession(s)
}

type Session struct {
	*ningen.State
	userID discord.Snowflake
}

var (
	_ cchat.Icon         = (*Session)(nil)
	_ cchat.Session      = (*Session)(nil)
	_ cchat.ServerList   = (*Session)(nil)
	_ cchat.SessionSaver = (*Session)(nil)
)

func NewSession(s *state.State) (*Session, error) {
	// Prefetch user.
	u, err := s.Me()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get current user")
	}

	n, err := ningen.FromState(s)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create a state wrapper")
	}

	return &Session{
		userID: u.ID,
		State:  n,
	}, nil
}

func (s *Session) ID() string {
	return s.userID.String()
}

func (s *Session) Name() text.Rich {
	u, err := s.Store.Me()
	if err != nil {
		// This shouldn't happen, ever.
		return text.Rich{Content: "<@" + s.userID.String() + ">"}
	}

	return text.Rich{Content: u.Username + "#" + u.Discriminator}
}

func (s *Session) Icon(ctx context.Context, iconer cchat.IconContainer) error {
	u, err := s.Store.Me()
	if err != nil {
		return errors.Wrap(err, "Failed to get the current user")
	}

	// Thanks to arikawa, AvatarURL is never empty.
	iconer.SetIcon(u.AvatarURL())
	return nil
}

func (s *Session) Servers(container cchat.ServersContainer) error {
	g, err := s.Guilds()
	if err != nil {
		return err
	}

	var servers = make([]cchat.Server, len(g))
	for i := range g {
		servers[i] = NewGuild(s, &g[i])
	}

	container.SetServers(servers)
	return nil
}

func (s *Session) Disconnect() error {
	return s.Close()
}

func (s *Session) Save() (map[string]string, error) {
	return map[string]string{
		"token": s.Token,
	}, nil
}
