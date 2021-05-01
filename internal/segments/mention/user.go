package mention

import (
	"bytes"
	"strings"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/arikawa/v2/state/store"
	"github.com/diamondburned/cchat-discord/internal/segments/colored"
	"github.com/diamondburned/cchat-discord/internal/segments/inline"
	"github.com/diamondburned/cchat-discord/internal/segments/segutil"
	"github.com/diamondburned/cchat-discord/internal/urlutils"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/ningen/v2"
)

// NewMemberText creates a new rich text describing the given member fetched
// from the state.
func NewMemberText(s *ningen.State, g discord.GuildID, u discord.UserID) text.Rich {
	m, err := s.Cabinet.Member(g, u)
	if err != nil {
		s.MemberState.RequestMember(g, u)
		return text.Plain(u.Mention())
	}

	user := NewUser(m.User)
	user.WithMember(*m)
	user.WithGuildID(g)
	user.WithState(s)
	user.Prefetch()

	rich := text.Rich{Content: user.DisplayName()}
	rich.Segments = []text.Segment{
		Segment{
			Start: 0,
			End:   len(rich.Content),
			User:  user,
		},
	}

	if m.User.Bot {
		rich.Content += " "
		rich.Segments = append(rich.Segments,
			colored.NewBlurple(segutil.Write(&rich, "[BOT]")),
		)
	}

	return rich
}

// NewUserText creates a new rich text describing a Discord user using the
// Presence API.
func NewUserText(s *ningen.State, u discord.UserID) text.Rich {
	var discordUser discord.User

	p, err := s.Presence(0, u)
	if err != nil {
		ready := s.Ready()
		if ready.User.ID != u {
			return text.Plain("Unknown user " + u.String())
		}

		discordUser = ready.User
	}

	user := NewUser(discordUser)
	if p != nil {
		user.WithPresence(*p)
	}
	user.WithState(s)
	user.Prefetch()

	rich := text.Rich{Content: user.DisplayName()}
	rich.Segments = []text.Segment{
		Segment{
			Start: 0,
			End:   len(rich.Content),
			User:  user,
		},
	}

	if discordUser.Bot {
		rich.Content += " "
		rich.Segments = append(rich.Segments,
			colored.NewBlurple(segutil.Write(&rich, "[BOT]")),
		)
	}

	return rich
}

type User struct {
	user    discord.User
	guildID discord.GuildID

	store  store.Cabinet
	ningen *ningen.State

	// optional prefetching

	guild    *discord.Guild
	member   *discord.Member
	presence *gateway.Presence

	color        uint32
	hasColor     bool
	fetchedColor bool
}

var (
	_ text.Colorer   = (*User)(nil)
	_ text.Avatarer  = (*User)(nil)
	_ text.Mentioner = (*User)(nil)
)

// NewUser creates a new user mention.
func NewUser(u discord.User) *User {
	return &User{
		user:  u,
		store: store.NoopCabinet,
	}
}

// User returns the internal user.
func (um *User) User() discord.User {
	return um.user
}

// UserID returns the user ID.
func (um *User) UserID() discord.UserID {
	return um.user.ID
}

// GuildID returns the guild ID, if any.
func (um *User) GuildID() discord.GuildID {
	return um.guildID
}

// SetGuildID sets the user's guild ID.
func (um *User) WithGuildID(guildID discord.GuildID) {
	um.guildID = guildID
}

// WithGuild sets the user's guild.
func (um *User) WithGuild(guild discord.Guild) {
	um.guildID = guild.ID
	um.guild = &guild
}

// WithMember sets the internal member to reduce roundtrips or cache hits. m can
// be nil.
func (um *User) WithMember(m discord.Member) {
	um.member = &m
}

// WithPresence sets the internal presence to reduce roundtrips or cache hits.
func (um *User) WithPresence(p gateway.Presence) {
	um.presence = &p
}

// WithState sets the internal state for usage.
func (um *User) WithState(state *ningen.State) {
	um.ningen = state
	um.store = state.Cabinet
}

// Prefetch prefetches everything in User.
func (um *User) Prefetch() {
	um.HasColor()
	um.getPresence()
}

// DisplayName returns either the nickname or the username.
func (um *User) DisplayName() string {
	if m := um.getMember(); m != nil && m.Nick != "" {
		return m.Nick
	}

	return um.user.Username
}

// HasColor returns true if the current user has a color.
func (um *User) HasColor() bool {
	if um.fetchedColor {
		return um.hasColor
	}

	// We don't have any member color if we have neither the member nor guild.
	if !um.guildID.IsValid() || !um.user.ID.IsValid() {
		um.fetchedColor = true
		return false
	}

	// We do have a valid GuildID, but the store might be a Noop, so we
	// shouldn't mark it as fetched.
	guild := um.getGuild()
	member := um.getMember()

	if guild == nil || member == nil {
		return false
	}

	um.fetchedColor = true
	um.color, um.hasColor = MemberColor(*guild, *member)

	return um.hasColor
}

func (um *User) Color() uint32 {
	if um.HasColor() {
		return text.SolidColor(um.color)
	}

	return colored.Blurple
}

func (um *User) AvatarSize() int {
	return 96
}

func (um *User) AvatarText() string {
	return um.DisplayName()
}

func (um *User) Avatar() (url string) {
	return urlutils.AvatarURL(um.user.AvatarURL())
}

func (um *User) MentionInfo() text.Rich {
	var content bytes.Buffer
	var segment text.Rich

	content.WriteString("Username: ")
	content.WriteString(um.user.Username)
	content.WriteByte('#')
	content.WriteString(um.user.Discriminator)
	content.WriteString("\n\n")

	// Write extra information if any, but only if we have the guild state.
	if um.guildID.IsValid() {
		guild := um.getGuild()
		member := um.getMember()

		if guild != nil && member != nil {
			// Write a prepended new line, as role writes will always prepend a
			// new line. This is to prevent a trailing new line.
			formatSectionf(&segment, &content, "Roles")

			for _, id := range member.RoleIDs {
				rl, ok := findRole(guild.Roles, id)
				if !ok {
					continue
				}

				// Prepend a new line before each item.
				content.WriteByte('\n')
				// Write exactly the role name, then grab the segment and color
				// it.
				start, end := segutil.WriteStringBuf(&content, "@"+rl.Name)
				// But we only add the color if the role has one.
				if rgb := rl.Color.Uint32(); rgb > 0 {
					segutil.Add(&segment, colored.NewSegment(start, end, rgb))
				}
			}

			// End section.
			content.WriteString("\n\n")
		}
	}

	// Does the user have rich presence? If so, write.
	if p := um.getPresence(); p != nil {
		for _, ac := range p.Activities {
			formatActivity(&segment, &content, ac)
			content.WriteString("\n\n")
		}
	}

	// These information can only be obtained from the state. As such, we check
	// if the state is given.
	if um.ningen != nil {
		// Write the user's note if any.
		formatSectionf(&segment, &content, "Note")
		content.WriteRune('\n')

		if note := um.ningen.NoteState.Note(um.user.ID); note != "" {
			start, end := segutil.WriteStringBuf(&content, note)
			segutil.Add(&segment, inline.NewSegment(start, end, text.AttributeMonospace))
		} else {
			start, end := segutil.WriteStringBuf(&content, "empty")
			segutil.Add(&segment, inline.NewSegment(start, end, text.AttributeDimmed))
		}

		content.WriteString("\n\n")
	}

	// Assign the written content into the text segment and return it after
	// trimming the trailing new line.
	segment.Content = strings.TrimSuffix(content.String(), "\n")
	return segment
}

func (um *User) getGuild() *discord.Guild {
	if um.guild != nil {
		return um.guild
	}

	g, err := um.store.Guild(um.guildID)
	if err != nil {
		return nil
	}

	um.guild = g
	return g
}

func (um *User) getMember() *discord.Member {
	// 0000 isn't a valid discriminator; the user is probably a webhook.
	// Fallback to the actual user.
	if !um.guildID.IsValid() || (um.user.Discriminator == "0000" && um.user.Bot) {
		return nil
	}

	if um.member != nil {
		return um.member
	}

	m, err := um.store.Member(um.guildID, um.user.ID)
	if err != nil {
		if um.ningen != nil {
			um.ningen.MemberState.RequestMember(um.guildID, um.user.ID)
		}

		return nil
	}

	um.user = m.User
	um.member = m
	return m
}

func (um *User) getPresence() *gateway.Presence {
	if um.presence != nil {
		return um.presence
	}

	p, err := um.store.Presence(um.guildID, um.user.ID)
	if err != nil {
		if um.guildID.IsValid() && um.ningen != nil {
			um.ningen.MemberState.RequestMember(um.guildID, um.user.ID)
		}

		return nil
	}

	um.presence = p
	return p
}
