package discord

import (
	"context"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/arikawa/state"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat/services"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/ningen"
	"github.com/pkg/errors"
)

func init() {
	services.RegisterService(&Service{})
}

// ErrInvalidSession is returned if SessionRestore is given a bad session.
var ErrInvalidSession = errors.New("invalid session")

type Service struct{}

var (
	_ cchat.Icon    = (*Service)(nil)
	_ cchat.Service = (*Service)(nil)
)

func (Service) Name() text.Rich {
	return text.Rich{Content: "Discord"}
}

func (Service) Icon(ctx context.Context, iconer cchat.IconContainer) (func(), error) {
	iconer.SetIcon("https://discord.com/assets/2c21aeda16de354ba5334551a883b481.png")
	return func() {}, nil
}

func (Service) Authenticate() cchat.Authenticator {
	return Authenticator{}
}

func (s Service) RestoreSession(data map[string]string) (cchat.Session, error) {
	tk, ok := data["token"]
	if !ok {
		return nil, ErrInvalidSession
	}

	return NewSessionToken(tk)
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
	return NewSessionToken(form[0])
}

type Session struct {
	*ningen.State
	userID discord.Snowflake
}

var (
	_ cchat.Icon         = (*Session)(nil)
	_ cchat.Session      = (*Session)(nil)
	_ cchat.SessionSaver = (*Session)(nil)
)

func NewSessionToken(token string) (*Session, error) {
	s, err := state.New(token)
	if err != nil {
		return nil, err
	}

	return NewSession(s)
}

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

	if err := s.Open(); err != nil {
		return nil, err
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

func (s *Session) Icon(ctx context.Context, iconer cchat.IconContainer) (func(), error) {
	u, err := s.Me()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get the current user")
	}

	// Thanks to arikawa, AvatarURL is never empty.
	iconer.SetIcon(AvatarURL(u.AvatarURL()))

	return s.AddHandler(func(u *gateway.UserUpdateEvent) {
		iconer.SetIcon(AvatarURL(u.AvatarURL()))
	}), nil
}

func (s *Session) Disconnect() error {
	return s.Close()
}

func (s *Session) Save() (map[string]string, error) {
	return map[string]string{
		"token": s.Token,
	}, nil
}

func (s *Session) Servers(container cchat.ServersContainer) error {
	switch {
	// If the user has guild folders:
	case len(s.Ready.Settings.GuildFolders) > 0:
		// TODO: account for missing guilds.
		var toplevels = make([]cchat.Server, 0, len(s.Ready.Settings.GuildFolders))

		for _, folder := range s.Ready.Settings.GuildFolders {
			// TODO: correct.
			switch {
			case folder.ID.Valid():
				fallthrough
			case len(folder.GuildIDs) > 1:
				toplevels = append(toplevels, NewGuildFolder(s, folder))

			case len(folder.GuildIDs) == 1:
				g, err := NewGuildFromID(s, folder.GuildIDs[0])
				if err != nil {
					return errors.Wrap(err, "Failed to get guild in folder")
				}
				toplevels = append(toplevels, g)
			}
		}

		container.SetServers(toplevels)

	// If the user doesn't have guild folders but has sorted their guilds
	// before:
	case len(s.Ready.Settings.GuildPositions) > 0:
		var guilds = make([]cchat.Server, 0, len(s.Ready.Settings.GuildPositions))

		for _, id := range s.Ready.Settings.GuildPositions {
			g, err := NewGuildFromID(s, id)
			if err != nil {
				return errors.Wrap(err, "Failed to get guild in position")
			}
			guilds = append(guilds, g)
		}

		container.SetServers(guilds)

	// None of the above:
	default:
		g, err := s.Guilds()
		if err != nil {
			return err
		}

		var servers = make([]cchat.Server, len(g))
		for i := range g {
			servers[i] = NewGuild(s, &g[i])
		}

		container.SetServers(servers)
	}

	return nil
}
