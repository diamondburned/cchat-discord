package discord

import (
	"github.com/diamondburned/arikawa/state"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat/text"
	"github.com/pkg/errors"
)

type Service struct{}

var (
	_ cchat.Service = (*Service)(nil)
	_ cchat.Icon    = (*Service)(nil)
)

func (Service) Name() text.Rich {
	return text.Rich{Content: "Discord"}
}

func (Service) Icon(iconer cchat.IconContainer) error {
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

	// Prefetch user.
	_, err = s.Me()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get current user")
	}

	return &Session{
		State: s,
	}, nil
}

type Session struct {
	*state.State
}

func (s *Session) ID() string {
	u, _ := s.Store.Me()
	return u.ID.String()
}

func (s *Session) Name() text.Rich {
	u, _ := s.Store.Me()
	return text.Rich{Content: u.Username + "#" + u.Discriminator}
}

func (s *Session) Icon(iconer cchat.IconContainer) error {
	u, _ := s.Store.Me()
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
