package hub

import (
	"context"
	"sync"
	"time"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/state"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/cchat/utils/empty"
	"github.com/pkg/errors"
)

// automatically add all channels with active messages within the past 5 days.
const autoAddActive = 5 * 24 * time.Hour

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
		switch channel.Type {
		case discord.DirectMessage, discord.GroupDM:
			// valid
		default:
			continue
		}

		if channelIsActive(s, channel, now) {
			ids[channel.ID] = struct{}{}
		}
	}

	return &activeList{active: ids}, nil
}

func channelIsActive(s *state.Instance, ch discord.Channel, now time.Time) bool {
	// Never show a muted channel, unless requested.
	muted := s.MutedState.Channel(ch.ID)
	if muted {
		return false
	}

	read := s.ReadState.FindLast(ch.ID)

	// recently created channel
	if ch.ID.Time().Add(autoAddActive).After(now) {
		return true
	}

	var lastMsg discord.MessageID
	if read != nil && read.LastMessageID.IsValid() {
		lastMsg = read.LastMessageID
	}
	if ch.LastMessageID > lastMsg {
		// We have a valid message ID in the read state and it is smaller than
		// the last message in the channel, so this channel is not read.
		if lastMsg.IsValid() {
			return true
		}

		lastMsg = ch.LastMessageID
	}

	// last message is recent
	if lastMsg.IsValid() && lastMsg.Time().Add(autoAddActive).After(now) {
		return true
	}

	return false
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

func (hub *Server) Name(_ context.Context, l cchat.LabelContainer) (func(), error) {
	l.SetLabel(text.Plain("Incoming Messages"))
	return func() {}, nil
}

func (hub *Server) Columnate() bool { return false }

// ActiveChannelIDs returns the list of active channel IDs, that is, the channel
// IDs that should be displayed separately.
func (hub *Server) ActiveChannelIDs() []discord.ChannelID {
	return hub.acList.list()
}

// Close unbinds the message handlers from the hub, invalidating it forever.
func (hub *Server) Close() { hub.msgs.cancel() }

func (hub *Server) AsMessenger() cchat.Messenger { return hub.msgs }
