package private

import (
	"context"
	"sort"
	"sync"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/session/channel"
	"github.com/diamondburned/cchat-discord/internal/discord/session/private/hub"
	"github.com/diamondburned/cchat-discord/internal/discord/state"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/cchat/utils/empty"
	"github.com/pkg/errors"
)

// I don't think the cchat specs said anything about sharing a cchat.Server, so
// we might need to do this. Nevertheless, it seems overkill.
type containerSet struct {
	mut sync.Mutex
	set map[cchat.ServersContainer]struct{}
}

func newContainerSet() *containerSet {
	return &containerSet{
		set: map[cchat.ServersContainer]struct{}{},
	}
}

func (cset *containerSet) Register(container cchat.ServersContainer) {
	cset.mut.Lock()
	cset.set[container] = struct{}{}
	cset.mut.Unlock()
}

// prependServer wraps around Server to always prepend this wrapped server on
// top of the servers container.
type prependServer struct{ cchat.Server }

var _ cchat.ServerUpdate = (*prependServer)(nil)

// PreviousID returns the appropriate parameters to prepend this server.
func (ps prependServer) PreviousID() (cchat.ID, bool) {
	// Return the private container's ID so this server goes right after it.
	return "!!!private-container!!!", false
}

func (cset *containerSet) AddChannel(s *state.Instance, ch *discord.Channel) {
	c, err := channel.New(s, *ch)
	if err != nil {
		return
	}

	replace := prependServer{Server: c}

	cset.mut.Lock()

	for container := range cset.set {
		container.UpdateServer(replace)
	}

	cset.mut.Unlock()
}

type Private struct {
	empty.Server
	state      *state.Instance
	hub        *hub.Server
	containers *containerSet
}

func New(s *state.Instance) (cchat.Server, error) {
	containers := newContainerSet()

	hubServer, err := hub.New(s, containers)
	if err != nil {
		return nil, errors.Wrap(err, "failed to make hub server")
	}

	return Private{
		state:      s,
		hub:        hubServer,
		containers: containers,
	}, nil
}

func (priv Private) ID() cchat.ID {
	// Not even a number, so no chance of colliding with snowflakes.
	return "!!!private-container!!!"
}

func (priv Private) Name(_ context.Context, l cchat.LabelContainer) (func(), error) {
	l.SetLabel(text.Plain("Private Channels"))
	return func() {}, nil
}

func (priv Private) AsLister() cchat.Lister { return priv }

type activeChannel struct {
	*discord.Channel
	*gateway.ReadState // used for sorting
}

func (active activeChannel) LastMessageID() discord.MessageID {
	if active.ReadState == nil {
		return active.Channel.LastMessageID
	}
	if active.ReadState.LastMessageID > active.Channel.LastMessageID {
		return active.ReadState.LastMessageID
	}
	if active.Channel.LastMessageID.IsValid() {
		return active.Channel.LastMessageID
	}
	// Whatever.
	return discord.MessageID(active.Channel.ID)
}

func (priv Private) Servers(container cchat.ServersContainer) (func(), error) {
	activeIDs := priv.hub.ActiveChannelIDs()

	channels := make([]activeChannel, 0, len(activeIDs))

	for _, id := range activeIDs {
		c, err := priv.state.Channel(id)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get private channel")
		}

		channels = append(channels, activeChannel{
			Channel:   c,
			ReadState: priv.state.ReadState.FindLast(id),
		})
	}

	// Sort so that channels with the largest last message ID (and therefore the
	// latest message) will be on top.
	sort.Slice(channels, func(i, j int) bool {
		return channels[i].LastMessageID() > channels[j].LastMessageID()
	})

	servers := make([]cchat.Server, len(channels)+1)
	servers[0] = priv.hub

	for i, ch := range channels {
		c, err := channel.New(priv.state, *ch.Channel)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create server for private channel")
		}

		servers[i+1] = c
	}

	container.SetServers(servers)
	priv.containers.Register(container)
	return func() {}, nil
}
