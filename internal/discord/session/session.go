package session

import (
	"context"
	"time"

	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/arikawa/v2/session"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/folder"
	"github.com/diamondburned/cchat-discord/internal/discord/guild"
	"github.com/diamondburned/cchat-discord/internal/discord/private"
	"github.com/diamondburned/cchat-discord/internal/discord/shared/state"
	"github.com/diamondburned/cchat-discord/internal/funcutil"
	"github.com/diamondburned/cchat-discord/internal/segments/mention"
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

func (s *Session) Name(ctx context.Context, l cchat.LabelContainer) (func(), error) {
	u, err := s.state.Cabinet.Me()
	if err != nil {
		l.SetLabel(text.Plain("<@" + s.state.UserID.String() + ">"))
	} else {
		user := mention.NewUser(*u)
		user.WithState(s.state.State)
		user.Prefetch()

		rich := text.Plain(user.DisplayName())
		rich.Segments = []text.Segment{
			mention.Segment{
				End:  len(rich.Content),
				User: user,
			},
		}

		l.SetLabel(rich)
	}

	// TODO.
	return func() {}, nil
}

func (s *Session) Disconnect() error {
	return s.state.CloseGracefully()
}

func (s *Session) AsSessionSaver() cchat.SessionSaver { return s.state }

func (s *Session) Servers(container cchat.ServersContainer) (func(), error) {
	if err := s.servers(container); err != nil {
		return nil, err
	}

	retryFn := func() {
		// We should set up a back-off here.
		for s.servers(container) != nil {
			time.Sleep(5 * time.Second)
		}
	}

	stop := funcutil.JoinCancels(
		// Reset the entire container when the session is closed.
		s.state.AddHandler(func(*session.Closed) {
			container.SetServers(nil)
		}),

		// Set the entire container again once reconnected.
		s.state.AddHandler(func(*ningen.Connected) {
			retryFn()
		}),

		// Update the entire container when we update the guild list. Blame
		// Discord on this one.
		s.state.AddHandler(func(update *gateway.UserSettingsUpdateEvent) {
			if update.GuildFolders != nil || update.GuildPositions != nil {
				retryFn()
			}
		}),
	)

	return stop, nil
}

func (s *Session) servers(container cchat.ServersContainer) error {
	ready := s.state.Ready()

	// If the user has guild folders:
	if len(ready.UserSettings.GuildFolders) > 0 {
		// TODO: account for missing guilds.
		toplevels := make([]cchat.Server, 1, len(ready.UserSettings.GuildFolders)+1)
		toplevels[0] = s.private

		for _, guildFolder := range ready.UserSettings.GuildFolders {
			// TODO: correct.
			// TODO: correct how? What did I mean by this?
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
		return nil
	}

	// If the user doesn't have guild folders but has sorted their guilds
	// before:
	if len(ready.UserSettings.GuildPositions) > 0 {
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
		return nil
	}

	// None of the above:
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
	return nil
}
