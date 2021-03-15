package labels

import (
	"sync"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/funcutil"
	"github.com/diamondburned/cchat-discord/internal/segments/mention"
	"github.com/diamondburned/ningen/v2"
)

// Repository is a repository containing LabelContainers. It watches for update
// events from the gateway.
//
// If a labeler that is already registered were to be added again, then the
// adder function will do nothing and will return a callback that does nothing.
type Repository struct {
	state   *ningen.State
	detach  func()
	stopped bool

	mutex  sync.Mutex
	stores labelContainers
}

// NewRepository creates a new repository.
func NewRepository(state *ningen.State) *Repository {
	r := Repository{
		state:  state,
		stores: newLabelContainers(),
	}

	r.detach = funcutil.JoinCancels(
		state.AddHandler(r.onGuildUpdate),
		state.AddHandler(r.onMemberUpdate),
		state.AddHandler(r.onMemberRemove),
		state.AddHandler(r.onChannelUpdate),
		state.AddHandler(r.onChannelDelete),
		state.AddHandler(r.onGuildMembersChunk),
		// TODO: *gateway.GuildMemberListUpdate
	)

	return &r
}

func (r *Repository) onGuildUpdate(ev *gateway.GuildUpdateEvent) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.stopped {
		return
	}

	guild, ok := r.stores.guilds[ev.ID]
	if !ok {
		return
	}

	rich := mention.NewGuildText(r.state, ev.ID)

	for labeler := range guild.guild {
		labeler.SetLabel(rich)
	}
}

// AddGuildLabel adds a label to display the given guild ID. Refer to Repository
// for more documentation.
func (r *Repository) AddGuildLabel(guildID discord.GuildID, l cchat.LabelContainer) func() {
	l.SetLabel(mention.NewGuildText(r.state, guildID))

	r.mutex.Lock()
	defer r.mutex.Unlock()

	guild, _ := r.stores.guilds[guildID]
	guild.guild.Add(l)
	r.stores.guilds[guildID] = guild

	return func() {
		r.mutex.Lock()
		defer r.mutex.Unlock()

		guild, _ := r.stores.guilds[guildID]
		guild.guild.Remove(l)

		if guild.IsEmpty() {
			delete(r.stores.guilds, guildID)
			return
		}

		r.stores.guilds[guildID] = guild
	}
}

func (r *Repository) onMemberRemove(ev *gateway.GuildMemberRemoveEvent) {
	// Not sure what to do.
}

func (r *Repository) onMemberUpdate(ev *gateway.GuildMemberUpdateEvent) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.stopped {
		return
	}

	guild, _ := r.stores.guilds[ev.GuildID]
	member, ok := guild.members[ev.User.ID]
	if !ok {
		return
	}

	rich := mention.NewMemberText(r.state, ev.GuildID, ev.User.ID)

	for labeler := range member {
		labeler.SetLabel(rich)
	}
}

func (r *Repository) onGuildMembersChunk(chunk *gateway.GuildMembersChunkEvent) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.stopped {
		return
	}

	guild, ok := r.stores.guilds[chunk.GuildID]
	if !ok {
		return
	}

	for _, member := range chunk.Members {
		m, ok := guild.members[member.User.ID]
		if ok {
			rich := mention.NewMemberText(r.state, chunk.GuildID, member.User.ID)

			for labeler := range m {
				labeler.SetLabel(rich)
			}
		}
	}
}

// AddMemberLabel adds a label to display the given member live. If the given
// guildID is not valid, then AddPresenceLabel will be called. Refer to
// Repository for more documentation.
func (r *Repository) AddMemberLabel(
	guildID discord.GuildID, userID discord.UserID, l cchat.LabelContainer) func() {

	if !guildID.IsValid() {
		return r.AddPresenceLabel(userID, l)
	}

	l.SetLabel(mention.NewMemberText(r.state, guildID, userID))

	r.mutex.Lock()
	defer r.mutex.Unlock()

	guild, _ := r.stores.guilds[guildID]

	llist := guild.members[userID]
	llist.Add(l)

	if guild.members == nil {
		guild.members = make(map[discord.UserID]labelerList, 1)
	}

	guild.members[userID] = llist
	r.stores.guilds[guildID] = guild

	return func() {
		r.mutex.Lock()
		defer r.mutex.Unlock()

		guild, _ := r.stores.guilds[guildID]

		llist := guild.members[userID]
		llist.Remove(l)

		if guild.IsEmpty() {
			delete(r.stores.guilds, guildID)
			return
		}

		guild.members[userID] = llist
		r.stores.guilds[guildID] = guild
	}
}

func (r *Repository) onChannelDelete(ev *gateway.ChannelDeleteEvent) {
	// Not sure what to do.
}

func (r *Repository) onChannelUpdate(ev *gateway.ChannelUpdateEvent) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.stopped {
		return
	}

	channel, ok := r.stores.channels[ev.ID]
	if !ok {
		return
	}

	rich := mention.NewChannelText(r.state, ev.ID)

	for labeler := range channel {
		labeler.SetLabel(rich)
	}
}

// AddChannelLabel adds a label to display the given channel live. Refer to
// Repository for more documentation.
func (r *Repository) AddChannelLabel(chID discord.ChannelID, l cchat.LabelContainer) func() {
	l.SetLabel(mention.NewChannelText(r.state, chID))

	r.mutex.Lock()
	defer r.mutex.Unlock()

	llist := r.stores.channels[chID]
	llist.Add(l)
	r.stores.channels[chID] = llist

	return func() {
		r.mutex.Lock()
		defer r.mutex.Unlock()

		llist := r.stores.channels[chID]
		llist.Remove(l)

		if len(llist) == 0 {
			delete(r.stores.channels, chID)
			return
		}

		r.stores.channels[chID] = llist
	}
}

func (r *Repository) AddPresenceLabel(uID discord.UserID, l cchat.LabelContainer) func() {
	// TODO: Presence update events
	// TODO: user fallbacks
	panic("Implement me")
	return nil
}

// Stop detaches all handlers.
func (r *Repository) Stop() {
	r.mutex.Lock()
	r.stopped = true
	r.mutex.Unlock()

	r.detach()
}
