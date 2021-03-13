package guild

import (
	"context"
	"sort"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/category"
	"github.com/diamondburned/cchat-discord/internal/discord/channel"
	"github.com/diamondburned/cchat-discord/internal/discord/state"
	"github.com/diamondburned/cchat-discord/internal/urlutils"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/cchat/utils/empty"
	"github.com/pkg/errors"
)

type Guild struct {
	empty.Server
	id    discord.GuildID
	state *state.Instance
}

func New(s *state.Instance, g *discord.Guild) cchat.Server {
	return &Guild{
		id:    g.ID,
		state: s,
	}
}

func NewFromID(s *state.Instance, gID discord.GuildID) (cchat.Server, error) {
	g, err := s.Cabinet.Guild(gID)
	if err != nil {
		return nil, err
	}

	return New(s, g), nil
}

func (g *Guild) self() (*discord.Guild, error) {
	return g.state.Cabinet.Guild(g.id)
}

func (g *Guild) ID() cchat.ID {
	return g.id.String()
}

func (g *Guild) Name() text.Rich {
	s, err := g.self()
	if err != nil {
		// This shouldn't happen.
		return text.Rich{Content: g.id.String()}
	}

	return text.Rich{Content: s.Name}
}

func (g *Guild) AsIconer() cchat.Iconer { return g }

func (g *Guild) Icon(ctx context.Context, iconer cchat.IconContainer) (func(), error) {
	s, err := g.self()
	if err != nil {
		// This shouldn't happen.
		return nil, errors.Wrap(err, "Failed to get guild")
	}

	// Used for comparison.
	if s.Icon != "" {
		iconer.SetIcon(urlutils.AvatarURL(s.IconURL()))
	}

	return g.state.AddHandler(func(update *gateway.GuildUpdateEvent) {
		if g.id == update.ID {
			iconer.SetIcon(urlutils.AvatarURL(s.IconURL()))
		}
	}), nil
}

func (g *Guild) AsLister() cchat.Lister { return g }

func (g *Guild) Servers(container cchat.ServersContainer) error {
	c, err := g.state.Channels(g.id)
	if err != nil {
		return errors.Wrap(err, "Failed to get channels")
	}

	// Only get top-level channels (those with category ID being null).
	var toplevels = category.FilterAccessible(g.state, category.FilterCategory(c, 0))

	// Sort so that positions are correct.
	sort.SliceStable(toplevels, func(i, j int) bool {
		return toplevels[i].Position < toplevels[j].Position
	})

	// Sort so that channels are before categories.
	sort.SliceStable(toplevels, func(i, _ int) bool {
		return toplevels[i].Type != discord.GuildCategory
	})

	chs := make([]cchat.Server, 0, len(toplevels))
	ids := make(map[discord.ChannelID]struct{}, len(toplevels))

	for _, ch := range toplevels {
		switch ch.Type {
		case discord.GuildCategory:
			chs = append(chs, category.New(g.state, ch))
		case discord.GuildText:
			c, err := channel.New(g.state, ch)
			if err != nil {
				return errors.Wrapf(err, "Failed to make channel %q: %v", ch.Name, err)
			}
			chs = append(chs, c)
		default:
			continue
		}
	}

	container.SetServers(chs)

	// TODO: account for insertion/deletion.

	return nil
}
