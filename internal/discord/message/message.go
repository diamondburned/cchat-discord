package message

import (
	"strings"
	"time"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/state"
	"github.com/diamondburned/cchat-discord/internal/segments"
	"github.com/diamondburned/cchat-discord/internal/segments/mention"
	"github.com/diamondburned/cchat-discord/internal/segments/reference"
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

func NewMessageUpdateContent(msg discord.Message, s *state.Instance) Message {
	// Check if content is empty.
	if msg.Content == "" {
		// Then grab the content from the state.
		m, err := s.Cabinet.Message(msg.ChannelID, msg.ID)
		if err == nil {
			msg.Content = m.Content
		}
	}

	var content = segments.ParseMessage(&msg, s.Cabinet)
	return Message{
		messageHeader: newHeader(msg),
		content:       content,
	}
}

func NewMessageUpdateAuthor(
	msg discord.Message, member discord.Member, g discord.Guild, s *state.Instance) Message {

	author := NewGuildMember(member, g, s)
	if ref := ReferencedMessage(msg, s, true); ref != nil {
		author.AddMessageReference(*ref, s)
	}

	return Message{
		messageHeader: newHeader(msg),
		author:        NewGuildMember(member, g, s),
	}
}

// NewGuildMessageCreate uses the session to create a message. It does not do
// API calls. Member is optional. This is the only call that populates the Nonce
// in the header.
func NewGuildMessageCreate(c *gateway.MessageCreateEvent, s *state.Instance) Message {
	// Copy and change the nonce.
	message := c.Message
	message.Nonce = s.Nonces.Load(c.Nonce)

	// This should not error.
	g, err := s.Cabinet.Guild(c.GuildID)
	if err != nil {
		return NewMessage(message, s, NewUser(c.Author, s))
	}

	if c.Member == nil {
		c.Member, _ = s.Cabinet.Member(c.GuildID, c.Author.ID)
	}
	if c.Member == nil {
		s.MemberState.RequestMember(c.GuildID, c.Author.ID)
		return NewMessage(message, s, NewUser(c.Author, s))
	}

	return NewMessage(message, s, NewGuildMember(*c.Member, *g, s))
}

// NewBacklogMessage uses the session to create a message fetched from the
// backlog. It takes in an existing guild and tries to fetch a new member, if
// it's nil.
func NewBacklogMessage(m discord.Message, s *state.Instance, g discord.Guild) Message {
	// If the message doesn't have a guild, then we don't need all the
	// complicated member fetching process.
	if !m.GuildID.IsValid() {
		return NewMessage(m, s, NewUser(m.Author, s))
	}

	mem, err := s.Cabinet.Member(m.GuildID, m.Author.ID)
	if err != nil {
		s.MemberState.RequestMember(m.GuildID, m.Author.ID)
		return NewMessage(m, s, NewUser(m.Author, s))
	}

	return NewMessage(m, s, NewGuildMember(*mem, g, s))
}

func NewDirectMessage(m discord.Message, s *state.Instance) Message {
	return NewMessage(m, s, NewUser(m.Author, s))
}

func NewMessage(m discord.Message, s *state.Instance, author Author) Message {
	var content text.Rich

	if ref := ReferencedMessage(m, s, true); ref != nil {
		// TODO: markup support
		var refmsg = "> " + ref.Content
		if len(refmsg) > 120 {
			refmsg = refmsg[:120] + "..."
		}

		content.Content = strings.ReplaceAll(refmsg, "\n", "  ") + "\n"
		content.Segments = []text.Segment{
			reference.NewMessageSegment(0, len(content.Content), ref.ID),
		}

		author.AddMessageReference(*ref, s)
	}

	// Render the message content.
	segments.ParseMessageRich(&content, &m, s.Cabinet)

	// Request members in mentions if we're in a guild.
	if m.GuildID.IsValid() {
		for _, segment := range content.Segments {
			mention, ok := segment.(*mention.Segment)
			if !ok {
				continue
			}

			// If this is not a user mention, then skip. If we already have a
			// member, then skip. We could check this using the timestamp, as we
			// might have a user set into the member field.
			if mention.User == nil || mention.User.Member.Joined.IsValid() {
				continue
			}

			// Request the member.
			s.MemberState.RequestMember(m.GuildID, mention.User.Member.User.ID)
		}
	}

	return Message{
		messageHeader: newHeaderNonce(m, m.Nonce),
		author:        author,
		content:       content,
	}
}

func (m Message) Author() cchat.Author {
	if !m.author.id.IsValid() {
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
