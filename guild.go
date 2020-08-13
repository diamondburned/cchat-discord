package discord

import (
	"context"
	"sort"
	"strings"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/segments"
	"github.com/diamondburned/cchat/text"
	"github.com/pkg/errors"
)

type GuildFolder struct {
	gateway.GuildFolder
	session *Session
}

var (
	_ cchat.Server     = (*Guild)(nil)
	_ cchat.ServerList = (*Guild)(nil)
)

func NewGuildFolder(s *Session, gf gateway.GuildFolder) *GuildFolder {
	// Name should never be empty.
	if gf.Name == "" {
		var names = make([]string, 0, len(gf.GuildIDs))

		for _, id := range gf.GuildIDs {
			if g, _ := s.Store.Guild(id); g != nil {
				names = append(names, g.Name)
			}
		}

		gf.Name = strings.Join(names, ", ")
	}

	return &GuildFolder{
		GuildFolder: gf,
		session:     s,
	}
}

func (gf *GuildFolder) ID() string {
	return gf.GuildFolder.ID.String()
}

func (gf *GuildFolder) Name() text.Rich {
	var name = text.Rich{
		// 1en space for style.
		Content: gf.GuildFolder.Name,
	}

	if gf.GuildFolder.Color > 0 {
		name.Segments = []text.Segment{
			// The length of this black box is actually 3. Mind == blown.
			segments.NewColored(len(name.Content), gf.GuildFolder.Color.Uint32()),
		}
	}

	return name
}

func (gf *GuildFolder) Servers(container cchat.ServersContainer) error {
	var servers = make([]cchat.Server, len(gf.GuildIDs))

	for i, id := range gf.GuildIDs {
		g, err := gf.session.Guild(id)
		if err != nil {
			continue
		}

		servers[i] = NewGuild(gf.session, g)
	}

	container.SetServers(servers)
	return nil
}

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

func NewGuildFromID(s *Session, gID discord.Snowflake) (*Guild, error) {
	g, err := s.Guild(gID)
	if err != nil {
		return nil, err
	}

	return NewGuild(s, g), nil
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

func (g *Guild) Icon(ctx context.Context, iconer cchat.IconContainer) (func(), error) {
	s, err := g.self(ctx)
	if err != nil {
		// This shouldn't happen.
		return nil, errors.Wrap(err, "Failed to get guild")
	}

	// Used for comparison.
	var hash = s.Icon
	if hash != "" {
		iconer.SetIcon(AvatarURL(s.IconURL()))
	}

	return g.session.AddHandler(func(g *gateway.GuildUpdateEvent) {
		if g.Icon != hash {
			hash = g.Icon
			iconer.SetIcon(AvatarURL(s.IconURL()))
		}
	}), nil
}

func (g *Guild) Servers(container cchat.ServersContainer) error {
	c, err := g.session.Channels(g.id)
	if err != nil {
		return errors.Wrap(err, "Failed to get channels")
	}

	// Only get top-level channels (those with category ID being null).
	var toplevels = filterAccessible(g.session, filterCategory(c, discord.NullSnowflake))

	// Sort so that positions are correct.
	sort.SliceStable(toplevels, func(i, j int) bool {
		return toplevels[i].Position < toplevels[j].Position
	})

	// Sort so that channels are before categories.
	sort.SliceStable(toplevels, func(i, _ int) bool {
		return toplevels[i].Type != discord.GuildCategory
	})

	var chs = make([]cchat.Server, 0, len(toplevels))

	for _, ch := range toplevels {
		switch ch.Type {
		case discord.GuildCategory:
			chs = append(chs, NewCategory(g.session, ch))
		case discord.GuildText:
			c, err := NewChannel(g.session, ch)
			if err != nil {
				return errors.Wrapf(err, "Failed to make channel %q: %v", ch.Name, err)
			}
			chs = append(chs, c)
		}
	}

	container.SetServers(chs)
	return nil
}
