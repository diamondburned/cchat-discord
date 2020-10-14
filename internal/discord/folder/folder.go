package folder

import (
	"strconv"
	"strings"

	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/guild"
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
			g, err := s.Store.Guild(id)
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

func (gf *GuildFolder) Name() text.Rich {
	var name = text.Rich{
		// 1en space for style.
		Content: gf.GuildFolder.Name,
	}

	if gf.GuildFolder.Color > 0 {
		name.Segments = []text.Segment{
			// The length of this black box is actually 3. Mind == blown.
			colored.New(len(name.Content), gf.GuildFolder.Color.Uint32()),
		}
	}

	return name
}

// IsLister returns true.
func (gf *GuildFolder) AsLister() cchat.Lister { return gf }

func (gf *GuildFolder) Servers(container cchat.ServersContainer) error {
	var servers = make([]cchat.Server, 0, len(gf.GuildIDs))

	for _, id := range gf.GuildIDs {
		g, err := gf.state.Guild(id)
		if err != nil {
			continue
		}

		servers = append(servers, guild.New(gf.state, g))
	}

	container.SetServers(servers)
	return nil
}
