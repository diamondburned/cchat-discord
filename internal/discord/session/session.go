package session

import (
	"context"

	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/arikawa/session"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/folder"
	"github.com/diamondburned/cchat-discord/internal/discord/guild"
	"github.com/diamondburned/cchat-discord/internal/discord/state"
	"github.com/diamondburned/cchat-discord/internal/urlutils"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/cchat/utils/empty"
	"github.com/diamondburned/ningen"
	"github.com/pkg/errors"
)

var ErrMFA = session.ErrMFA

type Session struct {
	empty.Session
	*state.Instance
}

func NewFromInstance(i *state.Instance) (cchat.Session, error) {
	return &Session{Instance: i}, nil
}

func (s *Session) ID() cchat.ID {
	return s.UserID.String()
}

func (s *Session) Name() text.Rich {
	u, err := s.Store.Me()
	if err != nil {
		// This shouldn't happen, ever.
		return text.Rich{Content: "<@" + s.UserID.String() + ">"}
	}

	return text.Rich{Content: u.Username + "#" + u.Discriminator}
}

func (s *Session) AsIconer() cchat.Iconer { return s }

func (s *Session) Icon(ctx context.Context, iconer cchat.IconContainer) (func(), error) {
	u, err := s.Me()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get the current user")
	}

	// Thanks to arikawa, AvatarURL is never empty.
	iconer.SetIcon(urlutils.AvatarURL(u.AvatarURL()))

	return s.AddHandler(func(*gateway.UserUpdateEvent) {
		// Bypass the event and use the state cache.
		if u, err := s.Store.Me(); err == nil {
			iconer.SetIcon(urlutils.AvatarURL(u.AvatarURL()))
		}
	}), nil
}

func (s *Session) Disconnect() error {
	return s.Close()
}

func (s *Session) AsSessionSaver() cchat.SessionSaver { return s.Instance }

func (s *Session) Servers(container cchat.ServersContainer) error {
	// Reset the entire container when the session is closed.
	s.AddHandler(func(*session.Closed) {
		container.SetServers(nil)
	})

	// Set the entire container again once reconnected.
	s.AddHandler(func(*ningen.Connected) {
		s.servers(container)
	})

	return s.servers(container)
}

func (s *Session) servers(container cchat.ServersContainer) error {
	switch {
	// If the user has guild folders:
	case len(s.Ready.Settings.GuildFolders) > 0:
		// TODO: account for missing guilds.
		var toplevels = make([]cchat.Server, 0, len(s.Ready.Settings.GuildFolders))

		for _, guildFolder := range s.Ready.Settings.GuildFolders {
			// TODO: correct.
			switch {
			case guildFolder.ID != 0:
				fallthrough
			case len(guildFolder.GuildIDs) > 1:
				toplevels = append(toplevels, folder.New(s.Instance, guildFolder))

			case len(guildFolder.GuildIDs) == 1:
				g, err := guild.NewFromID(s.Instance, guildFolder.GuildIDs[0])
				if err != nil {
					continue
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
			g, err := guild.NewFromID(s.Instance, id)
			if err != nil {
				continue
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
			servers[i] = guild.New(s.Instance, &g[i])
		}

		container.SetServers(servers)
	}

	return nil
}
