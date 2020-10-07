package message

import (
	"time"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/state"
	"github.com/diamondburned/cchat-discord/internal/segments"
	"github.com/diamondburned/cchat-discord/internal/segments/mention"
	"github.com/diamondburned/cchat/text"
)

type messageHeader struct {
	id        discord.MessageID
	time      discord.Timestamp
	channelID discord.ChannelID
	guildID   discord.GuildID
	nonce     string
}

var _ cchat.MessageHeader = (*messageHeader)(nil)

func newHeader(msg discord.Message) messageHeader {
	var h = messageHeader{
		id:        msg.ID,
		time:      msg.Timestamp,
		channelID: msg.ChannelID,
		guildID:   msg.GuildID,
		nonce:     msg.Nonce,
	}
	if msg.EditedTimestamp.IsValid() {
		h.time = msg.EditedTimestamp
	}
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
		m, err := s.Store.Message(msg.ChannelID, msg.ID)
		if err == nil {
			msg.Content = m.Content
		}
	}

	return Message{
		messageHeader: newHeader(msg),
		content:       segments.ParseMessage(&msg, s.Store),
	}
}

func NewMessageUpdateAuthor(
	msg discord.Message, member discord.Member, g discord.Guild, s *state.Instance) Message {

	return Message{
		messageHeader: newHeader(msg),
		author:        NewGuildMember(member, g, s),
	}
}

// NewMessageCreate uses the session to create a message. It does not do
// API calls. Member is optional.
func NewMessageCreate(c *gateway.MessageCreateEvent, s *state.Instance) Message {
	// This should not error.
	g, err := s.Store.Guild(c.GuildID)
	if err != nil {
		return NewMessage(c.Message, s, NewUser(c.Author, s))
	}

	if c.Member == nil {
		c.Member, _ = s.Store.Member(c.GuildID, c.Author.ID)
	}
	if c.Member == nil {
		s.MemberState.RequestMember(c.GuildID, c.Author.ID)
		return NewMessage(c.Message, s, NewUser(c.Author, s))
	}

	return NewMessage(c.Message, s, NewGuildMember(*c.Member, *g, s))
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

	mem, err := s.Store.Member(m.GuildID, m.Author.ID)
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
	// Render the message content.
	var content = segments.ParseMessage(&m, s.Store)

	// Request members in mentions if we're in a guild.
	if m.GuildID.IsValid() {
		for _, segment := range content.Segments {
			if mention, ok := segment.(*mention.Segment); ok {
				// If this is not a user mention, then skip.
				if mention.User == nil {
					continue
				}

				// If we already have a member, then skip. We could check this
				// using the timestamp, as we might have a user set into the
				// member field
				if mention.User.Member.Joined.IsValid() {
					continue
				}

				// Request the member.
				s.MemberState.RequestMember(m.GuildID, mention.User.Member.User.ID)
			}
		}
	}

	return Message{
		messageHeader: newHeader(m),
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
