package labels

import (
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/cchat"
)

type labelContainers struct {
	guilds   map[discord.GuildID]guildContainer
	channels map[discord.ChannelID]labelerList
	// presences map[discord.UserID]labelerList
}

func newLabelContainers() labelContainers {
	return labelContainers{
		guilds:   map[discord.GuildID]guildContainer{},
		channels: map[discord.ChannelID]labelerList{},
		// presences: map[discord.UserID]labelerList{},
	}
}

type guildContainer struct {
	guild   labelerList                    // optional
	members map[discord.UserID]labelerList // optional
}

// IsEmpty returns true if the container no longer holds any labeler.
func (gcont guildContainer) IsEmpty() bool {
	return len(gcont.guild) == 0 && len(gcont.members) == 0
}

// labelerList is a list of labelers.
type labelerList map[cchat.LabelContainer]struct{}

// Add adds the given labeler. If the map is nil, then a new one is created.
func (llist *labelerList) Add(l cchat.LabelContainer) {
	if *llist == nil {
		*llist = make(map[cchat.LabelContainer]struct{}, 1)
	}

	(*llist)[l] = struct{}{}
}

// Remove removes the given labeler.
func (llist labelerList) Remove(l cchat.LabelContainer) {
	delete(llist, l)
}
