package discord

import (
	"sort"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat/text"
	"github.com/pkg/errors"
)

type Category struct {
	id      discord.Snowflake
	guildID discord.Snowflake
	session *Session
}

var (
	_ cchat.Server     = (*Category)(nil)
	_ cchat.ServerList = (*Category)(nil)
)

func NewCategory(s *Session, ch discord.Channel) *Category {
	return &Category{
		id:      ch.ID,
		guildID: ch.GuildID,
		session: s,
	}
}

func (c *Category) ID() string {
	return c.id.String()
}

func (c *Category) Name() text.Rich {
	t, err := c.session.Channel(c.id)
	if err != nil {
		// This shouldn't happen.
		return text.Rich{Content: c.id.String()}
	}

	return text.Rich{
		Content: "⯆ " + t.Name,
	}
}

func (c *Category) Servers(container cchat.ServersContainer) error {
	t, err := c.session.Channels(c.guildID)
	if err != nil {
		return errors.Wrap(err, "Failed to get channels")
	}

	// Filter out channels with this category ID.
	var chs = filterAccessible(c.session, filterCategory(t, c.id))

	sort.Slice(chs, func(i, j int) bool {
		return chs[i].Position < chs[j].Position
	})

	var chv = make([]cchat.Server, len(chs))
	for i := range chs {
		chv[i] = NewChannel(c.session, chs[i])
	}

	container.SetServers(chv)
	return nil
}
