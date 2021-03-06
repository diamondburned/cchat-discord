package session

import (
	"context"

	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/arikawa/v2/session"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/folder"
	"github.com/diamondburned/cchat-discord/internal/discord/guild"
	"github.com/diamondburned/cchat-discord/internal/discord/private"
	"github.com/diamondburned/cchat-discord/internal/discord/state"
	"github.com/diamondburned/cchat-discord/internal/urlutils"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/cchat/utils/empty"
	"github.com/diamondburned/ningen/v2"
	"github.com/pkg/errors"
)

var ErrMFA = session.ErrMFA

type Session struct {
	empty.Session
	private cchat.Server
	state   *state.Instance
}

func NewFromInstance(i *state.Instance) (cchat.Session, error) {
	priv, err := private.New(i)
	if err != nil {
		return nil, errors.Wrap(err, "failed to make main private server")
	}

	return &Session{
		private: priv,
		state:   i,
	}, nil
}

func (s *Session) ID() cchat.ID {
	return s.state.UserID.String()
}

func (s *Session) Name() text.Rich {
	u, err := s.state.Cabinet.Me()
	if err != nil {
		// This shouldn't happen, ever.
		return text.Rich{Content: "<@" + s.state.UserID.String() + ">"}
	}

	return text.Rich{Content: u.Username + "#" + u.Discriminator}
}

func (s *Session) AsIconer() cchat.Iconer { return s }

func (s *Session) Icon(ctx context.Context, iconer cchat.IconContainer) (func(), error) {
	u, err := s.state.Me()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get the current user")
	}

	// Thanks to arikawa, AvatarURL is never empty.
	iconer.SetIcon(urlutils.AvatarURL(u.AvatarURL()))

	return s.state.AddHandler(func(*gateway.UserUpdateEvent) {
		// Bypass the event and use the state cache.
		if u, err := s.state.Cabinet.Me(); err == nil {
			iconer.SetIcon(urlutils.AvatarURL(u.AvatarURL()))
		}
	}), nil
}

func (s *Session) Disconnect() error {
	return s.state.Close()
}

func (s *Session) AsSessionSaver() cchat.SessionSaver { return s.state }

func (s *Session) Servers(container cchat.ServersContainer) error {
	// Reset the entire container when the session is closed.
	s.state.AddHandler(func(*session.Closed) {
		container.SetServers(nil)
	})

	// Set the entire container again once reconnected.
	s.state.AddHandler(func(*ningen.Connected) {
		s.servers(container)
	})

	return s.servers(container)
}

func (s *Session) servers(container cchat.ServersContainer) error {
	ready := s.state.Ready()

	switch {
	// If the user has guild folders:
	case len(ready.UserSettings.GuildFolders) > 0:
		// TODO: account for missing guilds.
		toplevels := make([]cchat.Server, 1, len(ready.UserSettings.GuildFolders)+1)
		toplevels[0] = s.private

		for _, guildFolder := range ready.UserSettings.GuildFolders {
			// TODO: correct.
			switch {
			case guildFolder.ID != 0:
				fallthrough
			case len(guildFolder.GuildIDs) > 1:
				toplevels = append(toplevels, folder.New(s.state, guildFolder))

			case len(guildFolder.GuildIDs) == 1:
				g, err := guild.NewFromID(s.state, guildFolder.GuildIDs[0])
				if err != nil {
					continue
				}
				toplevels = append(toplevels, g)
			}
		}

		container.SetServers(toplevels)

	// If the user doesn't have guild folders but has sorted their guilds
	// before:
	case len(ready.UserSettings.GuildPositions) > 0:
		guilds := make([]cchat.Server, 1, len(ready.UserSettings.GuildPositions)+1)
		guilds[0] = s.private

		for _, id := range ready.UserSettings.GuildPositions {
			g, err := guild.NewFromID(s.state, id)
			if err != nil {
				continue
			}
			guilds = append(guilds, g)
		}

		container.SetServers(guilds)

	// None of the above:
	default:
		g, err := s.state.Guilds()
		if err != nil {
			return err
		}

		servers := make([]cchat.Server, len(g)+1)
		servers[0] = s.private

		for i := range g {
			servers[i+1] = guild.New(s.state, &g[i])
		}

		container.SetServers(servers)
	}

	return nil
}
