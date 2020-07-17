package discord

import (
	"context"
	"sort"
	"time"

	"github.com/diamondburned/arikawa/api"
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/segments"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/ningen/states/read"
	"github.com/pkg/errors"
)

func chGuildCheck(chType discord.ChannelType) bool {
	switch chType {
	case discord.GuildCategory, discord.GuildText:
		return true
	default:
		return false
	}
}

func filterAccessible(s *Session, chs []discord.Channel) []discord.Channel {
	filtered := chs[:0]

	for _, ch := range chs {
		p, err := s.Permissions(ch.ID, s.userID)
		// Treat error as non-fatal and add the channel anyway.
		if err != nil || p.Has(discord.PermissionViewChannel) {
			filtered = append(filtered, ch)
		}
	}

	return filtered
}

func filterCategory(chs []discord.Channel, catID discord.Snowflake) []discord.Channel {
	var filtered = chs[:0]
	var catvalid = catID.Valid()

	for _, ch := range chs {
		switch {
		// If the given ID is not valid, then we look for channels with
		// similarly invalid category IDs, because yes, Discord really sends
		// inconsistent responses.
		case !catvalid && !ch.CategoryID.Valid():
			fallthrough
		// Basic comparison.
		case ch.CategoryID == catID:
			if chGuildCheck(ch.Type) {
				filtered = append(filtered, ch)
			}
		}
	}

	return filtered
}

type Channel struct {
	id      discord.Snowflake
	guildID discord.Snowflake
	session *Session
}

var (
	_ cchat.Server                        = (*Channel)(nil)
	_ cchat.ServerMessage                 = (*Channel)(nil)
	_ cchat.ServerMessageSender           = (*Channel)(nil)
	_ cchat.ServerMessageAttachmentSender = (*Channel)(nil)
	_ cchat.ServerMessageSendCompleter    = (*Channel)(nil)
	_ cchat.ServerNickname                = (*Channel)(nil)
	_ cchat.ServerMessageEditor           = (*Channel)(nil)
	_ cchat.ServerMessageActioner         = (*Channel)(nil)
	_ cchat.ServerMessageTypingIndicator  = (*Channel)(nil)
	_ cchat.ServerMessageUnreadIndicator  = (*Channel)(nil)
)

func NewChannel(s *Session, ch discord.Channel) *Channel {
	return &Channel{
		id:      ch.ID,
		guildID: ch.GuildID,
		session: s,
	}
}

// self does not do IO.
func (ch *Channel) self() (*discord.Channel, error) {
	return ch.session.Store.Channel(ch.id)
}

// messages does not do IO.
func (ch *Channel) messages() ([]discord.Message, error) {
	return ch.session.Store.Messages(ch.id)
}

func (ch *Channel) guild() (*discord.Guild, error) {
	if ch.guildID.Valid() {
		return ch.session.Guild(ch.guildID)
	}
	return nil, errors.New("channel not in a guild")
}

func (ch *Channel) ID() string {
	return ch.id.String()
}

func (ch *Channel) Name() text.Rich {
	c, err := ch.self()
	if err != nil {
		return text.Rich{Content: ch.id.String()}
	}

	if c.NSFW {
		return text.Rich{Content: "#!" + c.Name}
	} else {
		return text.Rich{Content: "#" + c.Name}
	}
}

func (ch *Channel) Nickname(ctx context.Context, labeler cchat.LabelContainer) (func(), error) {
	// We don't have a nickname if we're not in a guild.
	if !ch.guildID.Valid() {
		return func() {}, nil
	}

	state := ch.session.WithContext(ctx)

	// MemberColor should fill up the state cache.
	c, err := state.MemberColor(ch.guildID, ch.session.userID)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get self member color")
	}

	m, err := state.Member(ch.guildID, ch.session.userID)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get self member")
	}

	var rich = text.Rich{Content: m.User.Username}
	if m.Nick != "" {
		rich.Content = m.Nick
	}
	if c > 0 {
		rich.Segments = []text.Segment{
			segments.NewColored(len(rich.Content), c.Uint32()),
		}
	}

	labeler.SetLabel(rich)

	// Copy the user ID to use.
	var selfID = m.User.ID

	return ch.session.AddHandler(func(g *gateway.GuildMemberUpdateEvent) {
		if g.GuildID != ch.guildID || g.User.ID != selfID {
			return
		}

		var rich = text.Rich{Content: m.User.Username}
		if m.Nick != "" {
			rich.Content = m.Nick
		}

		c, err := ch.session.MemberColor(g.GuildID, selfID)
		if err == nil {
			rich.Segments = []text.Segment{
				segments.NewColored(len(rich.Content), c.Uint32()),
			}
		}

		labeler.SetLabel(rich)
	}), nil
}

func (ch *Channel) JoinServer(ctx context.Context, ct cchat.MessagesContainer) (func(), error) {
	state := ch.session.WithContext(ctx)

	m, err := state.Messages(ch.id)
	if err != nil {
		return nil, err
	}

	var addcancel = newCancels()

	var constructor func(discord.Message) cchat.MessageCreate

	if ch.guildID.Valid() {
		// Create the backlog without any member information.
		g, err := state.Guild(ch.guildID)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to get guild")
		}

		constructor = func(m discord.Message) cchat.MessageCreate {
			return NewBacklogMessage(m, ch.session, *g)
		}

		// Subscribe to typing events.
		ch.session.MemberState.Subscribe(ch.guildID)

		// Listen to new members before creating the backlog and requesting members.
		addcancel(ch.session.AddHandler(func(c *gateway.GuildMembersChunkEvent) {
			if c.GuildID != ch.guildID {
				return
			}

			m, err := ch.messages()
			if err != nil {
				// TODO: log
				return
			}

			g, err := ch.guild()
			if err != nil {
				return
			}

			// Loop over all messages and replace the author. The latest
			// messages are in front.
			for _, msg := range m {
				for _, member := range c.Members {
					if msg.Author.ID != member.User.ID {
						continue
					}

					ct.UpdateMessage(NewMessageUpdateAuthor(msg, member, *g))
				}
			}
		}))
	} else {
		constructor = func(m discord.Message) cchat.MessageCreate {
			return NewDirectMessage(m, ch.session)
		}
	}

	// Only do all this if we even have any messages.
	if len(m) > 0 {
		// Sort messages chronologically using the ID so that the oldest messages
		// (ones with the smallest snowflake) is in front.
		sort.Slice(m, func(i, j int) bool { return m[i].ID < m[j].ID })

		// Iterate from the earliest messages to the latest messages.
		for _, m := range m {
			ct.CreateMessage(constructor(m))
		}

		// Mark this channel as read.
		ch.session.ReadState.MarkRead(ch.id, m[len(m)-1].ID)
	}

	// Bind the handler.
	addcancel(
		ch.session.AddHandler(func(m *gateway.MessageCreateEvent) {
			if m.ChannelID == ch.id {
				ct.CreateMessage(NewMessageCreate(m, ch.session))
				ch.session.ReadState.MarkRead(ch.id, m.ID)
			}
		}),
		ch.session.AddHandler(func(m *gateway.MessageUpdateEvent) {
			// If the updated content is empty. TODO: add embed support.
			if m.ChannelID == ch.id {
				ct.UpdateMessage(NewMessageUpdateContent(m.Message, ch.session))
			}
		}),
		ch.session.AddHandler(func(m *gateway.MessageDeleteEvent) {
			if m.ChannelID == ch.id {
				ct.DeleteMessage(NewHeaderDelete(m))
			}
		}),
	)

	return joinCancels(addcancel()), nil
}

func (ch *Channel) SendMessage(msg cchat.SendableMessage) error {
	var send = api.SendMessageData{Content: msg.Content()}
	if noncer, ok := msg.(cchat.MessageNonce); ok {
		send.Nonce = noncer.Nonce()
	}
	if attcher, ok := msg.(cchat.SendableMessageAttachments); ok {
		send.Files = addAttachments(attcher.Attachments())
	}

	_, err := ch.session.SendMessageComplex(ch.id, send)
	return err
}

func (ch *Channel) SendAttachments(atts []cchat.MessageAttachment) error {
	_, err := ch.session.SendMessageComplex(ch.id, api.SendMessageData{
		Files: addAttachments(atts),
	})
	return err
}

func addAttachments(atts []cchat.MessageAttachment) []api.SendMessageFile {
	var files = make([]api.SendMessageFile, len(atts))
	for i, a := range atts {
		files[i] = api.SendMessageFile{
			Name:   a.Name,
			Reader: a,
		}
	}
	return files
}

// MessageEditable returns true if the given message ID belongs to the current
// user.
func (ch *Channel) MessageEditable(id string) bool {
	s, err := discord.ParseSnowflake(id)
	if err != nil {
		return false
	}

	m, err := ch.session.Store.Message(ch.id, s)
	if err != nil {
		return false
	}

	return m.Author.ID == ch.session.userID
}

// RawMessageContent returns the raw message content from Discord.
func (ch *Channel) RawMessageContent(id string) (string, error) {
	s, err := discord.ParseSnowflake(id)
	if err != nil {
		return "", errors.Wrap(err, "Failed to parse ID")
	}

	m, err := ch.session.Store.Message(ch.id, s)
	if err != nil {
		return "", errors.Wrap(err, "Failed to get the message")
	}

	return m.Content, nil
}

// EditMessage edits the message to the given content string.
func (ch *Channel) EditMessage(id, content string) error {
	s, err := discord.ParseSnowflake(id)
	if err != nil {
		return errors.Wrap(err, "Failed to parse ID")
	}

	_, err = ch.session.EditText(ch.id, s, content)
	return err
}

const (
	ActionDelete = "Delete"
)

var ErrUnknownAction = errors.New("unknown message action")

func (ch *Channel) DoMessageAction(action, id string) error {
	s, err := discord.ParseSnowflake(id)
	if err != nil {
		return errors.Wrap(err, "Failed to parse ID")
	}

	switch action {
	case ActionDelete:
		return ch.session.DeleteMessage(ch.id, s)
	default:
		return ErrUnknownAction
	}
}

func (ch *Channel) MessageActions(id string) []string {
	s, err := discord.ParseSnowflake(id)
	if err != nil {
		return nil
	}

	m, err := ch.session.Store.Message(ch.id, s)
	if err != nil {
		return nil
	}

	// Get the current user.
	u, err := ch.session.Store.Me()
	if err != nil {
		return nil
	}

	// Can we have delete? We can if this is our own message.
	var canDelete = m.Author.ID == u.ID

	// We also can if we have the Manage Messages permission, which would allow
	// us to delete others' messages.
	if !canDelete {
		canDelete = ch.canManageMessages(u.ID)
	}

	if canDelete {
		return []string{ActionDelete}
	}

	return []string{}
}

// canManageMessages returns whether or not the user is allowed to manage
// messages.
func (ch *Channel) canManageMessages(userID discord.Snowflake) bool {
	// If we're not in a guild, then clearly we cannot.
	if !ch.guildID.Valid() {
		return false
	}

	// We need the guild, member and channel to calculate the permission
	// overrides.

	g, err := ch.guild()
	if err != nil {
		return false
	}

	c, err := ch.self()
	if err != nil {
		return false
	}

	m, err := ch.session.Store.Member(ch.guildID, userID)
	if err != nil {
		return false
	}

	p := discord.CalcOverwrites(*g, *c, *m)
	// The Manage Messages permission allows the user to delete others'
	// messages, so we'll return true if that is the case.
	return p.Has(discord.PermissionManageMessages)
}

// CompleteMessage implements message input completion capability for Discord.
// This method supports user mentions, channel mentions and emojis.
//
// For the individual implementations, refer to channel_completion.go.
func (ch *Channel) CompleteMessage(words []string, i int) (entries []cchat.CompletionEntry) {
	var word = words[i]
	// Word should have at least a character for the char check.
	if len(word) < 1 {
		return
	}

	switch word[0] {
	case '@':
		return ch.completeMentions(word[1:])
	case '#':
		return ch.completeChannels(word[1:])
	case ':':
		return ch.completeEmojis(word[1:])
	}

	return
}

func (ch *Channel) Typing() error {
	return ch.session.Typing(ch.id)
}

// TypingTimeout returns 10 seconds.
func (ch *Channel) TypingTimeout() time.Duration {
	return 10 * time.Second
}

func (ch *Channel) TypingSubscribe(ti cchat.TypingIndicator) (func(), error) {
	return ch.session.AddHandler(func(t *gateway.TypingStartEvent) {
		// Ignore channel mismatch or if the typing event is ours.
		if t.ChannelID != ch.id || t.UserID == ch.session.userID {
			return
		}
		if typer, err := NewTyper(ch.session.Store, t); err == nil {
			ti.AddTyper(typer)
		}
	}), nil
}

func (ch *Channel) UnreadIndicate(indicator cchat.UnreadIndicator) (func(), error) {
	if rs := ch.session.ReadState.FindLast(ch.id); rs != nil {
		c, err := ch.self()
		if err != nil {
			return nil, errors.Wrap(err, "Failed to get self channel")
		}

		if c.LastMessageID > rs.LastMessageID {
			indicator.SetUnread(true, rs.MentionCount > 0)
		}
	}

	return ch.session.ReadState.OnUpdate(func(ev *read.UpdateEvent) {
		if ch.id == ev.ChannelID {
			indicator.SetUnread(ev.Unread, ev.MentionCount > 0)
		}
	}), nil
}

func newCancels() func(...func()) []func() {
	var cancels []func()
	return func(appended ...func()) []func() {
		cancels = append(cancels, appended...)
		return cancels
	}
}

func joinCancels(cancellers []func()) func() {
	return func() {
		for _, c := range cancellers {
			c()
		}
	}
}
