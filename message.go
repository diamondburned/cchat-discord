package discord

import (
	"time"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/segments"
	"github.com/diamondburned/cchat-discord/urlutils"
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

// AvatarURL wraps the URL with URL queries for the avatar.
func AvatarURL(URL string) string {
	return urlutils.AvatarURL(URL)
}

type Author struct {
	id     discord.Snowflake
	name   text.Rich
	avatar string
}

func NewUser(u discord.User) Author {
	var name = text.Rich{Content: u.Username}
	if u.Bot {
		name.Content += " "
		name.Segments = append(name.Segments,
			segments.NewBlurpleSegment(segments.Write(&name, "[BOT]")),
		)
	}

	return Author{
		id:     u.ID,
		name:   name,
		avatar: AvatarURL(u.AvatarURL()),
	}
}

func NewGuildMember(m discord.Member, g discord.Guild) Author {
	return Author{
		id:     m.User.ID,
		name:   RenderMemberName(m, g),
		avatar: AvatarURL(m.User.AvatarURL()),
	}
}

func RenderMemberName(m discord.Member, g discord.Guild) text.Rich {
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

	if m.User.Bot {
		name.Content += " "
		name.Segments = append(name.Segments,
			segments.NewBlurpleSegment(segments.Write(&name, "[BOT]")),
		)
	}

	return name
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

func NewMessageUpdateContent(msg discord.Message, s *Session) Message {
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

func NewMessageUpdateAuthor(msg discord.Message, member discord.Member, g discord.Guild) Message {
	return Message{
		messageHeader: newHeader(msg),
		author:        NewGuildMember(member, g),
	}
}

// NewMessageCreate uses the session to create a message. It does not do
// API calls. Member is optional.
func NewMessageCreate(c *gateway.MessageCreateEvent, s *Session) Message {
	// This should not error.
	g, err := s.Store.Guild(c.GuildID)
	if err != nil {
		return NewMessage(c.Message, s, NewUser(c.Author))
	}

	if c.Member == nil {
		c.Member, _ = s.Store.Member(c.GuildID, c.Author.ID)
	}
	if c.Member == nil {
		s.MemberState.RequestMember(c.GuildID, c.Author.ID)
		return NewMessage(c.Message, s, NewUser(c.Author))
	}

	return NewMessage(c.Message, s, NewGuildMember(*c.Member, *g))
}

// NewBacklogMessage uses the session to create a message fetched from the
// backlog. It takes in an existing guild and tries to fetch a new member, if
// it's nil.
func NewBacklogMessage(m discord.Message, s *Session, g discord.Guild) Message {
	// If the message doesn't have a guild, then we don't need all the
	// complicated member fetching process.
	if !m.GuildID.Valid() {
		return NewMessage(m, s, NewUser(m.Author))
	}

	mem, err := s.Store.Member(m.GuildID, m.Author.ID)
	if err != nil {
		s.MemberState.RequestMember(m.GuildID, m.Author.ID)
		return NewMessage(m, s, NewUser(m.Author))
	}

	return NewMessage(m, s, NewGuildMember(*mem, g))
}

func NewDirectMessage(m discord.Message, s *Session) Message {
	return NewMessage(m, s, NewUser(m.Author))
}

func NewMessage(m discord.Message, s *Session, author Author) Message {
	return Message{
		messageHeader: newHeader(m),
		author:        author,
		content:       segments.ParseMessage(&m, s.Store),
	}
}

func (m Message) Author() cchat.MessageAuthor {
	if !m.author.id.Valid() {
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
