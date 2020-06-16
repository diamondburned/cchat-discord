package discord

import (
	"context"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat/text"
	"github.com/pkg/errors"
)

type Guild struct {
	id      discord.Snowflake
	session *Session
}

var (
	_ cchat.Icon       = (*Guild)(nil)
	_ cchat.Server     = (*Guild)(nil)
	_ cchat.ServerList = (*Guild)(nil)
)

func NewGuild(s *Session, g *discord.Guild) *Guild {
	return &Guild{
		id:      g.ID,
		session: s,
	}
}

func (g *Guild) self(ctx context.Context) (*discord.Guild, error) {
	return g.session.WithContext(ctx).Guild(g.id)
}

func (g *Guild) selfState() (*discord.Guild, error) {
	return g.session.Store.Guild(g.id)
}

func (g *Guild) ID() string {
	return g.id.String()
}

func (g *Guild) Name() text.Rich {
	s, err := g.selfState()
	if err != nil {
		// This shouldn't happen.
		return text.Rich{Content: g.id.String()}
	}

	return text.Rich{Content: s.Name}
}

func (g *Guild) Icon(ctx context.Context, iconer cchat.IconContainer) error {
	s, err := g.self(ctx)
	if err != nil {
		// This shouldn't happen.
		return errors.Wrap(err, "Failed to get guild")
	}

	if s.Icon != "" {
		iconer.SetIcon(s.IconURL() + "?size=64")
	}
	return nil
}

func (g *Guild) Servers(container cchat.ServersContainer) error {
	c, err := g.session.Channels(g.id)
	if err != nil {
		return errors.Wrap(err, "Failed to get channels")
	}

	var channels = make([]cchat.Server, len(c))
	for i := range c {
		channels[i] = NewChannel(g.session, c[i])
	}

	container.SetServers(channels)
	return nil
}
