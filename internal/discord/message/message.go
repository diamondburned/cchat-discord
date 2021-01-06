package message

import (
	"log"
	"strings"
	"time"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/state"
	"github.com/diamondburned/cchat-discord/internal/segments"
	"github.com/diamondburned/cchat-discord/internal/segments/inline"
	"github.com/diamondburned/cchat-discord/internal/segments/mention"
	"github.com/diamondburned/cchat-discord/internal/segments/reference"
	"github.com/diamondburned/cchat-discord/internal/segments/segutil"
	"github.com/diamondburned/cchat/text"
)

type messageHeader struct {
	id        discord.MessageID
	time      discord.Timestamp
	nonce     string
	channelID discord.ChannelID
	guildID   discord.GuildID
}

var _ cchat.MessageHeader = (*messageHeader)(nil)

func newHeader(msg discord.Message) messageHeader {
	return messageHeader{
		id:        msg.ID,
		time:      msg.Timestamp,
		channelID: msg.ChannelID,
		guildID:   msg.GuildID,
	}
}

func newHeaderNonce(msg discord.Message, nonce string) messageHeader {
	h := newHeader(msg)
	h.nonce = nonce
	return h
}

func NewHeaderDelete(d *gateway.MessageDeleteEvent) messageHeader {
	return messageHeader{
		id:        d.ID,
		time:      discord.Timestamp(time.Now()),
		channelID: d.ChannelID,
		guildID:   d.GuildID,
	}
}

func (m messageHeader) ID() cchat.ID {
	return m.id.String()
}

func (m messageHeader) Nonce() string { return m.nonce }

func (m messageHeader) MessageID() discord.MessageID { return m.id }
func (m messageHeader) ChannelID() discord.ChannelID { return m.channelID }
func (m messageHeader) GuildID() discord.GuildID     { return m.guildID }

func (m messageHeader) Time() time.Time {
	return m.time.Time()
}

type Message struct {
	messageHeader

	author  Author
	content text.Rich

	// TODO
	mentioned bool
}

var (
	_ cchat.MessageCreate = (*Message)(nil)
	_ cchat.MessageUpdate = (*Message)(nil)
	_ cchat.MessageDelete = (*Message)(nil)
	_ cchat.Noncer        = (*Message)(nil)
)

// NewGuildMessageCreate uses the session to create a message. It does not do
// API calls. Member is optional. This is the only call that populates the Nonce
// in the header.
func NewGuildMessageCreate(c *gateway.MessageCreateEvent, s *state.Instance) Message {
	// Copy and change the nonce.
	message := c.Message
	message.Nonce = s.Nonces.Load(c.Nonce)

	user := mention.NewUser(c.Author)
	user.WithState(s.State)
	user.WithGuildID(c.GuildID)

	if c.Member != nil {
		user.WithMember(*c.Member)
	}

	user.Prefetch()

	return NewMessage(message, s, NewAuthor(user))
}

// NewBacklogMessage uses the session to create a message fetched from the
// backlog. It takes in an existing guild and tries to fetch a new member, if
// it's nil.
func NewBacklogMessage(m discord.Message, s *state.Instance) Message {
	// If the message doesn't have a guild, then we don't need all the
	// complicated member fetching process.
	if !m.GuildID.IsValid() {
		return NewDirectMessage(m, s)
	}

	user := mention.NewUser(m.Author)
	user.WithGuildID(m.GuildID)
	user.WithState(s.State)
	user.Prefetch()

	return NewMessage(m, s, NewAuthor(user))
}

// NewDirectMessage creates a new direct message.
func NewDirectMessage(m discord.Message, s *state.Instance) Message {
	user := mention.NewUser(m.Author)
	user.WithState(s.State)
	user.Prefetch()

	return NewMessage(m, s, NewAuthor(user))
}

// NewAuthorUpdate creates a new message that contains a new author.
func NewAuthorUpdate(msg discord.Message, m discord.Member, s *state.Instance) Message {
	user := mention.NewUser(msg.Author)
	user.WithState(s.State)
	user.WithGuildID(msg.GuildID)
	user.WithMember(m)

	author := NewAuthor(user)
	if ref := ReferencedMessage(msg, s, true); ref != nil {
		author.AddMessageReference(*ref, s)
	}

	return Message{
		messageHeader: newHeader(msg),
		author:        author,
	}
}

// NewContentUpdate creates a new message that does not have an author. It
// should be used for UpdateMessage only.
func NewContentUpdate(msg discord.Message, s *state.Instance) Message {
	// Check if content is empty.
	if msg.Content == "" {
		// Then grab the content from the state.
		m, err := s.Cabinet.Message(msg.ChannelID, msg.ID)
		if err == nil {
			msg = *m
		}
	}

	return newMessageContent(&msg, s)
}

// NewMessage creates a new message from the given author. It may modify author
// to add a message reference.
func NewMessage(m discord.Message, s *state.Instance, author Author) Message {
	message := newMessageContent(&m, s)
	message.author = author

	if m.ReferencedMessage != nil {
		message.author.AddMessageReference(*m.ReferencedMessage, s)
	}

	return message
}

// newMessageContent creates a new message with a content only. The given
// message will have its ReferencedMessage field validated and filled if
// available.
func newMessageContent(m *discord.Message, s *state.Instance) Message {
	// Ensure the validity of ReferencedMessage.
	m.ReferencedMessage = ReferencedMessage(*m, s, true)

	var content text.Rich

	switch m.Type {
	case discord.ChannelPinnedMessage:
		writeSegmented(&content, "Pinned ", "a message", " to this channel.",
			func(i, j int) text.Segment {
				if m.ReferencedMessage == nil {
					return nil
				}
				return reference.NewMessageSegment(i, j, m.ReferencedMessage.ID)
			},
		)

	case discord.GuildMemberJoinMessage:
		content.Content = "Joined the server."

	case discord.CallMessage:
		content.Content = "Calling you."

	case discord.ChannelIconChangeMessage:
		content.Content = "Changed the channel icon."

	case discord.ChannelNameChangeMessage:
		writeSegmented(&content, "Changed the channel name to ", m.Content, ".",
			func(i, j int) text.Segment {
				return mention.Segment{
					Start:   i,
					End:     j,
					Channel: mention.NewChannelFromID(s.State, m.ChannelID),
				}
			},
		)

	case discord.RecipientAddMessage:
		if len(m.Mentions) == 0 {
			content.Content = "Added recipient to the group."
			break
		}

		writeSegmented(&content,
			"Added ", m.Mentions[0].Username, " to the group.",
			segmentFuncFromMention(*m, s),
		)

	case discord.RecipientRemoveMessage:
		if len(m.Mentions) == 0 {
			content.Content = "Removed recipient from the group."
			break
		}

		writeSegmented(&content,
			"Removed ", m.Mentions[0].Username, " from the group.",
			segmentFuncFromMention(*m, s),
		)

	case discord.NitroBoostMessage:
		content.Content = "Boosted the server."
	case discord.NitroTier1Message:
		content.Content = "The server is now Nitro Boosted to Tier 1."
	case discord.NitroTier2Message:
		content.Content = "The server is now Nitro Boosted to Tier 2."
	case discord.NitroTier3Message:
		content.Content = "The server is now Nitro Boosted to Tier 3."

	case discord.ChannelFollowAddMessage:
		log.Printf("[Discord] Unknown message type: %#v\n", m)
		content.Content = "Type = discord.ChannelFollowAddMessage"

	case discord.GuildDiscoveryDisqualifiedMessage:
		log.Printf("[Discord] Unknown message type: %#v\n", m)
		content.Content = "Type = discord.GuildDiscoveryDisqualifiedMessage"

	case discord.GuildDiscoveryRequalifiedMessage:
		log.Printf("[Discord] Unknown message type: %#v\n", m)
		content.Content = "Type = discord.GuildDiscoveryRequalifiedMessage"

	case discord.ApplicationCommandMessage:
		fallthrough
	case discord.InlinedReplyMessage:
		fallthrough
	case discord.DefaultMessage:
		fallthrough
	default:
		return newRegularContent(*m, s)
	}

	segutil.Add(&content, inline.NewSegment(
		0, len(content.Content),
		text.AttributeDimmed|text.AttributeItalics,
	))

	return Message{
		messageHeader: newHeaderNonce(*m, m.Nonce),
		content:       content,
	}
}

func newRegularContent(m discord.Message, s *state.Instance) Message {
	var content text.Rich

	if m.ReferencedMessage != nil {
		refContent := []byte(m.ReferencedMessage.Content)
		segments.ParseWithMessageRich(&content, refContent, &m, s.Cabinet)

		content = segments.Ellipsize(content, 100)
		content.Content += "\n"

		segutil.Add(&content,
			reference.NewMessageSegment(0, len(content.Content)-1, m.ReferencedMessage.ID),
		)
	}

	segments.ParseMessageRich(&content, &m, s.Cabinet)

	return Message{
		messageHeader: newHeaderNonce(m, m.Nonce),
		content:       content,
	}
}

func (m Message) Author() cchat.Author {
	if m.author.user == nil {
		return nil
	}
	return m.author
}

func (m Message) Content() text.Rich {
	return m.content
}

func (m Message) Nonce() string {
	return m.nonce
}

func (m Message) Mentioned() bool {
	return m.mentioned
}

// ReferencedMessage searches for the referenced message if needed.
func ReferencedMessage(m discord.Message, s *state.Instance, wait bool) (reply *discord.Message) {
	// Deleted or does not exist.
	if m.Reference == nil || !m.Reference.MessageID.IsValid() {
		return nil
	}

	// Check these in case.
	if !m.Reference.ChannelID.IsValid() {
		m.Reference.ChannelID = m.ChannelID
	}
	if !m.Reference.GuildID.IsValid() {
		m.Reference.GuildID = m.GuildID
	}

	if m.ReferencedMessage != nil {
		// Set these in case Discord acts dumb.
		m.ReferencedMessage.GuildID = m.Reference.GuildID
		m.ReferencedMessage.ChannelID = m.Reference.ChannelID
		return m.ReferencedMessage
	}

	if !wait {
		reply, _ = s.Cabinet.Message(m.Reference.ChannelID, m.Reference.MessageID)
	} else {
		reply, _ = s.Message(m.Reference.ChannelID, m.Reference.MessageID)
	}

	return
}

// segmentFuncFromMention returns a function that gets the message's first
// mention and returns a segment created from it. It returns nil if the message
// does not have any mentions.
func segmentFuncFromMention(m discord.Message, s *state.Instance) func(i, j int) text.Segment {
	return func(i, j int) text.Segment {
		if len(m.Mentions) == 0 {
			return nil
		}

		firstMention := m.Mentions[0]

		user := mention.NewUser(firstMention.User)
		user.WithGuildID(m.GuildID)
		user.WithState(s.State)

		if firstMention.Member != nil {
			user.WithMember(*firstMention.Member)
		}

		user.Prefetch()

		return mention.NewSegment(i, j, user)
	}
}

func writeSegmented(rich *text.Rich, start, mid, end string, f func(i, j int) text.Segment) {
	var builder strings.Builder

	builder.WriteString(start)
	i, j := segutil.WriteStringBuilder(&builder, start)
	builder.WriteString(end)

	rich.Content = builder.String()

	if seg := f(i, j); seg != nil {
		rich.Segments = append(rich.Segments, f(i, j))
	}
}
