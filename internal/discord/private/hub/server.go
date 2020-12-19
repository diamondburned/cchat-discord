package hub

import (
	"sync"
	"time"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/state"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/cchat/utils/empty"
	"github.com/pkg/errors"
)

// automatically add all channels with active messages within the past 48 hours.
const autoAddActive = 24 * time.Hour

// activeList contains a list of channel IDs that should be put into its own
// channels.
type activeList struct {
	mut    sync.Mutex
	active map[discord.ChannelID]struct{}
}

func makeActiveList(s *state.Instance) (*activeList, error) {
	channels, err := s.PrivateChannels()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get private channels")
	}

	ids := make(map[discord.ChannelID]struct{}, len(channels))
	now := time.Now()

	for _, channel := range channels {
		if channel.LastMessageID.Time().Add(autoAddActive).After(now) {
			ids[channel.ID] = struct{}{}
		}
	}

	return &activeList{active: ids}, nil
}

func (acList *activeList) list() []discord.ChannelID {
	acList.mut.Lock()
	defer acList.mut.Unlock()

	var channelIDs = make([]discord.ChannelID, 0, len(acList.active))
	for channelID := range acList.active {
		channelIDs = append(channelIDs, channelID)
	}

	return channelIDs
}

func (acList *activeList) isActive(channelID discord.ChannelID) bool {
	acList.mut.Lock()
	defer acList.mut.Unlock()

	_, ok := acList.active[channelID]
	return ok
}

func (acList *activeList) add(chID discord.ChannelID) (changed bool) {
	acList.mut.Lock()
	defer acList.mut.Unlock()

	if _, ok := acList.active[chID]; ok {
		return false
	}

	acList.active[chID] = struct{}{}
	return true
}

// Server is the server (channel) that contains all incoming DM messages that
// are not being listened.
type Server struct {
	empty.Server
	acList *activeList
	msgs   *Messages
}

func New(s *state.Instance, adder ChannelAdder) (*Server, error) {
	acList, err := makeActiveList(s)
	if err != nil {
		return nil, errors.Wrap(err, "failed to make active guild list")
	}

	return &Server{
		acList: acList,
		msgs:   NewMessages(s, acList, adder),
	}, nil
}

func (hub *Server) ID() cchat.ID { return "!!!hub-server!!!" }

func (hub *Server) Name() text.Rich { return text.Plain("Incoming Messages") }

// ActiveChannelIDs returns the list of active channel IDs, that is, the channel
// IDs that should be displayed separately.
func (hub *Server) ActiveChannelIDs() []discord.ChannelID {
	return hub.acList.list()
}

// Close unbinds the message handlers from the hub, invalidating it forever.
func (hub *Server) Close() { hub.msgs.cancel() }

func (hub *Server) AsMessenger() cchat.Messenger { return hub.msgs }
