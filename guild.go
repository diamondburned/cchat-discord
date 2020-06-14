package discord

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat/text"
)

type Guild struct {
	id      discord.Snowflake
	name    string
	session *Session
}

var (
	_ cchat.Server = (*Guild)(nil)
)

func NewGuild(s *Session, g *discord.Guild) *Guild {
	return &Guild{
		id:      g.ID,
		name:    g.Name,
		session: s,
	}
}

func (g *Guild) ID() string {
	return g.id.String()
}

func (g *Guild) Name() text.Rich {
	return text.Rich{Content: g.name}
}

func (g *Guild) Guilds(container cchat.ServersContainer) error {
	return nil
}
