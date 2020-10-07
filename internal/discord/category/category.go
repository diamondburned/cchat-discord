package category

import (
	"sort"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/channel"
	"github.com/diamondburned/cchat-discord/internal/discord/state"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/cchat/utils/empty"
	"github.com/pkg/errors"
)

func ChGuildCheck(chType discord.ChannelType) bool {
	switch chType {
	case discord.GuildCategory, discord.GuildText:
		return true
	default:
		return false
	}
}

func FilterAccessible(s *state.Instance, chs []discord.Channel) []discord.Channel {
	filtered := chs[:0]

	for _, ch := range chs {
		p, err := s.Permissions(ch.ID, s.UserID)
		// Treat error as non-fatal and add the channel anyway.
		if err != nil || p.Has(discord.PermissionViewChannel) {
			filtered = append(filtered, ch)
		}
	}

	return filtered
}

func FilterCategory(chs []discord.Channel, catID discord.ChannelID) []discord.Channel {
	var filtered = chs[:0]
	var catvalid = catID.IsValid()

	for _, ch := range chs {
		switch {
		// If the given ID is not valid, then we look for channels with
		// similarly invalid category IDs, because yes, Discord really sends
		// inconsistent responses.
		case !catvalid && !ch.CategoryID.IsValid():
			fallthrough
		// Basic comparison.
		case ch.CategoryID == catID:
			if ChGuildCheck(ch.Type) {
				filtered = append(filtered, ch)
			}
		}
	}

	return filtered
}

type Category struct {
	empty.Server
	id      discord.ChannelID
	guildID discord.GuildID
	state   *state.Instance
}

func New(s *state.Instance, ch discord.Channel) cchat.Server {
	return &Category{
		id:      ch.ID,
		guildID: ch.GuildID,
		state:   s,
	}
}

func (c *Category) ID() cchat.ID {
	return c.id.String()
}

func (c *Category) Name() text.Rich {
	t, err := c.state.Channel(c.id)
	if err != nil {
		// This shouldn't happen.
		return text.Rich{Content: c.id.String()}
	}

	return text.Rich{
		Content: t.Name,
	}
}

func (c *Category) AsLister() cchat.Lister { return c }

func (c *Category) Servers(container cchat.ServersContainer) error {
	t, err := c.state.Channels(c.guildID)
	if err != nil {
		return errors.Wrap(err, "Failed to get channels")
	}

	// Filter out channels with this category ID.
	var chs = FilterAccessible(c.state, FilterCategory(t, c.id))

	sort.Slice(chs, func(i, j int) bool {
		return chs[i].Position < chs[j].Position
	})

	var chv = make([]cchat.Server, len(chs))
	for i := range chs {
		c, err := channel.New(c.state, chs[i])
		if err != nil {
			return errors.Wrapf(err, "Failed to make channel %s: %v", chs[i].Name, err)
		}

		chv[i] = c
	}

	container.SetServers(chv)
	return nil
}
