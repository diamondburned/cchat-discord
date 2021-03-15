package folder

import (
	"context"
	"strconv"
	"strings"

	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/session/guild"
	"github.com/diamondburned/cchat-discord/internal/discord/state"
	"github.com/diamondburned/cchat-discord/internal/segments/colored"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/cchat/utils/empty"
)

type GuildFolder struct {
	empty.Server
	gateway.GuildFolder
	state *state.Instance
}

func New(s *state.Instance, gf gateway.GuildFolder) cchat.Server {
	// Name should never be empty.
	if gf.Name == "" {
		var names = make([]string, 0, len(gf.GuildIDs))

		for _, id := range gf.GuildIDs {
			g, err := s.Cabinet.Guild(id)
			if err == nil {
				names = append(names, g.Name)
			}
		}

		gf.Name = strings.Join(names, ", ")
	}

	return &GuildFolder{
		GuildFolder: gf,
		state:       s,
	}
}

func (gf *GuildFolder) ID() cchat.ID {
	return strconv.FormatInt(int64(gf.GuildFolder.ID), 10)
}

func (gf *GuildFolder) Name(ctx context.Context, l cchat.LabelContainer) (func(), error) {
	var name = text.Rich{
		Content: gf.GuildFolder.Name,
	}

	if gf.GuildFolder.Color > 0 {
		name.Segments = []text.Segment{
			colored.New(len(name.Content), gf.GuildFolder.Color.Uint32()),
		}
	}

	// TODO: add folder updater from setting update events.
	return func() {}, nil
}

// IsLister returns true.
func (gf *GuildFolder) AsLister() cchat.Lister { return gf }

func (gf *GuildFolder) Servers(container cchat.ServersContainer) (func(), error) {
	var servers = make([]cchat.Server, 0, len(gf.GuildIDs))

	for _, id := range gf.GuildIDs {
		g, err := gf.state.Cabinet.Guild(id)
		if err != nil {
			continue
		}

		servers = append(servers, guild.New(gf.state, g))
	}

	container.SetServers(servers)

	// Return an empty callback. We're lazily redoing the whole list when a
	// guild moves for now.
	return func() {}, nil
}
