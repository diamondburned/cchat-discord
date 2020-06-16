package discord

import (
	"time"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/segments"
	"github.com/diamondburned/cchat/text"
)

type messageHeader struct {
	id        discord.Snowflake
	time      discord.Timestamp
	channelID discord.Snowflake
	guildID   discord.Snowflake
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
	if msg.EditedTimestamp.Valid() {
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

func (m messageHeader) ID() string {
	return m.id.String()
}

func (m messageHeader) Time() time.Time {
	return m.time.Time()
}

type Author struct {
	id     discord.Snowflake
	name   text.Rich
	avatar string
}

func NewUser(u discord.User) Author {
	return Author{
		id:     u.ID,
		name:   text.Rich{Content: u.Username},
		avatar: u.AvatarURL() + "?size=128",
	}
}

func NewGuildMember(m discord.Member, g discord.Guild) Author {
	var name = text.Rich{
		Content: m.User.Username,
	}

	// Update the nickname.
	if m.Nick != "" {
		name.Content = m.Nick
	}

	// Update the color.
	if c := discord.MemberColor(g, m); c > 0 {
		name.Segments = []text.Segment{
			segments.NewColored(len(name.Content), c.Uint32()),
		}
	}

	return Author{
		id:     m.User.ID,
		name:   name,
		avatar: m.User.AvatarURL() + "?size=128",
	}
}

func (a Author) ID() string {
	return a.id.String()
}

func (a Author) Name() text.Rich {
	return a.name
}

func (a Author) Avatar() string {
	return a.avatar
}

type Message struct {
	messageHeader

	author  Author
	content text.Rich

	// TODO
	mentioned bool
}

func NewMessageUpdateContent(msg discord.Message) Message {
	return Message{
		messageHeader: newHeader(msg),
		content:       text.Rich{Content: msg.Content},
	}
}

func NewMessageUpdateAuthor(msg discord.Message, member discord.Member, g discord.Guild) Message {
	return Message{
		messageHeader: newHeader(msg),
		author:        NewGuildMember(member, g),
	}
}

// NewMessageWithSession uses the session to create a message. It does not do
// API calls. Member is optional.
func NewMessageWithMember(m discord.Message, s *Session, mem *discord.Member) Message {
	// This should not error.
	g, err := s.Store.Guild(m.GuildID)
	if err != nil {
		return NewMessage(m, NewUser(m.Author))
	}

	if mem == nil {
		mem, _ = s.Store.Member(m.GuildID, m.Author.ID)
	}
	if mem == nil {
		s.Members.RequestMember(m.GuildID, m.Author.ID)
		return NewMessage(m, NewUser(m.Author))
	}

	return NewMessage(m, NewGuildMember(*mem, *g))
}

// NewBacklogMessage uses the session to create a message fetched from the
// backlog. It takes in an existing guild and tries to fetch a new member, if
// it's nil.
func NewBacklogMessage(m discord.Message, s *Session, g discord.Guild) Message {
	// If the message doesn't have a guild, then we don't need all the
	// complicated member fetching process.
	if !m.GuildID.Valid() {
		return NewMessage(m, NewUser(m.Author))
	}

	mem, err := s.Store.Member(m.GuildID, m.Author.ID)
	if err != nil {
		s.Members.RequestMember(m.GuildID, m.Author.ID)
		return NewMessage(m, NewUser(m.Author))
	}

	return NewMessage(m, NewGuildMember(*mem, g))
}

func NewDirectMessage(m discord.Message) Message {
	return NewMessage(m, NewUser(m.Author))
}

func NewMessage(m discord.Message, author Author) Message {
	return Message{
		messageHeader: newHeader(m),
		author:        author,
		content:       text.Rich{Content: m.Content},
	}
}

func (m Message) Author() cchat.MessageAuthor {
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
