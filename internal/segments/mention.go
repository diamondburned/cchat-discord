package segments

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/state"
	"github.com/diamondburned/cchat-discord/internal/urlutils"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/ningen"
	"github.com/diamondburned/ningen/md"
	"github.com/yuin/goldmark/ast"
)

// NameSegment represents a clickable member name; it does not implement colors.
type NameSegment struct {
	start, end int

	guild  discord.Guild
	member discord.Member
	state  *ningen.State // optional
}

var (
	_ text.Segment         = (*NameSegment)(nil)
	_ text.Mentioner       = (*NameSegment)(nil)
	_ text.MentionerAvatar = (*NameSegment)(nil)
)

func UserSegment(start, end int, u discord.User) NameSegment {
	return NameSegment{
		start:  start,
		end:    end,
		member: discord.Member{User: u},
	}
}

func MemberSegment(start, end int, guild discord.Guild, m discord.Member) NameSegment {
	return NameSegment{
		start:  start,
		end:    end,
		guild:  guild,
		member: m,
	}
}

// WithState assigns a ningen state into the given name segment. This allows the
// popovers to have additional information such as user notes.
func (m *NameSegment) WithState(state *ningen.State) {
	m.state = state
}

func (m NameSegment) Bounds() (start, end int) {
	return m.start, m.end
}

func (m NameSegment) MentionInfo() text.Rich {
	return userInfo(m.guild, m.member, m.state)
}

// Avatar returns the large avatar URL.
func (m NameSegment) Avatar() string {
	return m.member.User.AvatarURL()
}

type MentionSegment struct {
	start, end int
	*md.Mention

	store state.Store
	guild discord.GuildID
}

var (
	_ text.Segment         = (*MentionSegment)(nil)
	_ text.Colorer         = (*MentionSegment)(nil)
	_ text.Mentioner       = (*MentionSegment)(nil)
	_ text.MentionerAvatar = (*MentionSegment)(nil)
)

func (r *TextRenderer) mention(n *md.Mention, enter bool) ast.WalkStatus {
	if enter {
		var seg = MentionSegment{
			Mention: n,
			store:   r.store,
			guild:   r.msg.GuildID,
		}

		switch {
		case n.Channel != nil:
			seg.start, seg.end = r.writeString("#" + n.Channel.Name)
		case n.GuildUser != nil:
			seg.start, seg.end = r.writeString("@" + n.GuildUser.Username)
		case n.GuildRole != nil:
			seg.start, seg.end = r.writeString("@" + n.GuildRole.Name)
		default:
			// Unexpected error; skip.
			return ast.WalkSkipChildren
		}

		r.append(seg)
	}

	return ast.WalkContinue
}

func (m MentionSegment) Bounds() (start, end int) {
	return m.start, m.end
}

// Color tries to return the color of the mention segment, or it returns the
// usual blurple if none.
func (m MentionSegment) Color() (color uint32) {
	// Try digging through what we have for a color.
	switch {
	case m.GuildUser != nil && m.GuildUser.Member != nil:
		g, err := m.store.Guild(m.guild)
		if err != nil {
			return blurple
		}

		color = discord.MemberColor(*g, *m.GuildUser.Member).Uint32()

	case m.GuildRole != nil && m.GuildRole.Color > 0:
		color = m.GuildRole.Color.Uint32()
	}

	if color > 0 {
		return
	}

	return blurple
}

// TODO
func (m MentionSegment) MentionInfo() text.Rich {
	switch {
	case m.Channel != nil:
		return m.channelInfo()
	case m.GuildUser != nil:
		return m.userInfo()
	case m.GuildRole != nil:
		return m.roleInfo()
	}

	// Unknown; return an empty text.
	return text.Rich{}
}

// Avatar returns the user avatar if any, else it returns an empty URL.
func (m MentionSegment) Avatar() string {
	if m.GuildUser != nil {
		return m.GuildUser.AvatarURL()
	}

	return ""
}

func (m MentionSegment) channelInfo() text.Rich {
	var topic = m.Channel.Topic
	if m.Channel.NSFW {
		topic = "(NSFW)\n" + topic
	}

	if topic == "" {
		return text.Rich{}
	}

	return Parse([]byte(topic))
}

func (m MentionSegment) userInfo() text.Rich {
	if m.GuildUser.Member == nil {
		m.GuildUser.Member = &discord.Member{
			User: m.GuildUser.User,
		}
	}

	// Get the guild for the role slice. If not, then too bad.
	g, err := m.store.Guild(m.guild)
	if err != nil {
		g = &discord.Guild{}
	}

	return userInfo(*g, *m.GuildUser.Member, nil)
}

func (m MentionSegment) roleInfo() text.Rich {
	// // We don't have much to write here.
	// var segment = text.Rich{
	// 	Content: m.GuildRole.Name,
	// }

	// // Maybe add a color if we have any.
	// if c := m.GuildRole.Color.Uint32(); c > 0 {
	// 	segment.Segments = []text.Segment{
	// 		NewColored(len(m.GuildRole.Name), m.GuildRole.Color.Uint32()),
	// 	}
	// }

	return text.Rich{}
}

type LargeActivityImage struct {
	start int
	url   string
	text  string
}

func NewLargeActivityImage(start int, ac discord.Activity) LargeActivityImage {
	var text = ac.Assets.LargeText
	if text == "" {
		text = "Activity Image"
	}

	return LargeActivityImage{
		start: start,
		url:   urlutils.AssetURL(ac.ApplicationID, ac.Assets.LargeImage),
		text:  ac.Assets.LargeText,
	}
}

func (i LargeActivityImage) Bounds() (start, end int) { return i.start, i.start }
func (i LargeActivityImage) Image() string            { return i.url }
func (i LargeActivityImage) ImageSize() (w, h int)    { return 60, 60 }
func (i LargeActivityImage) ImageText() string        { return i.text }

func userInfo(guild discord.Guild, member discord.Member, state *ningen.State) text.Rich {
	var content bytes.Buffer
	var segment text.Rich

	// Write the username if the user has a nickname.
	if member.Nick != "" {
		content.WriteString("Username: ")
		content.WriteString(member.User.Username)
		content.WriteByte('#')
		content.WriteString(member.User.Discriminator)
		content.WriteString("\n\n")
	}

	// Write extra information if any, but only if we have the guild state.
	if len(member.RoleIDs) > 0 && guild.ID.IsValid() {
		// Write a prepended new line, as role writes will always prepend a new
		// line. This is to prevent a trailing new line.
		formatSectionf(&segment, &content, "Roles")

		for _, id := range member.RoleIDs {
			rl, ok := findRole(guild.Roles, id)
			if !ok {
				continue
			}

			// Prepend a new line before each item.
			content.WriteByte('\n')
			// Write exactly the role name, then grab the segment and color it.
			start, end := writestringbuf(&content, "@"+rl.Name)
			// But we only add the color if the role has one.
			if color := rl.Color.Uint32(); color > 0 {
				segmentadd(&segment, NewColoredSegment(start, end, rl.Color.Uint32()))
			}
		}

		// End section.
		content.WriteString("\n\n")
	}

	// These information can only be obtained from the state. As such, we check
	// if the state is given.
	if state != nil {
		// Does the user have rich presence? If so, write.
		if p, err := state.Presence(guild.ID, member.User.ID); err == nil {
			for _, ac := range p.Activities {
				formatActivity(&segment, &content, ac)
				content.WriteString("\n\n")
			}
		} else if guild.ID.IsValid() {
			// If we're still in a guild, then we can ask Discord for that
			// member with their presence attached.
			state.MemberState.RequestMember(guild.ID, member.User.ID)
		}

		// Write the user's note if any.
		if note := state.NoteState.Note(member.User.ID); note != "" {
			formatSectionf(&segment, &content, "Note")
			content.WriteRune('\n')

			start, end := writestringbuf(&content, note)
			segmentadd(&segment, InlineSegment{start, end, text.AttrMonospace})

			content.WriteString("\n\n")
		}
	}

	// Assign the written content into the text segment and return it after
	// trimming the trailing new line.
	segment.Content = strings.TrimSuffix(content.String(), "\n")
	return segment
}

func formatSectionf(segment *text.Rich, content *bytes.Buffer, f string, argv ...interface{}) {
	// Treat f as a regular string at first.
	var str = fmt.Sprintf("%s", f)

	// If there are argvs, then treat f as a format string.
	if len(argv) > 0 {
		str = fmt.Sprintf(str, argv...)
	}

	start, end := writestringbuf(content, str)
	segmentadd(segment, InlineSegment{start, end, text.AttrBold | text.AttrUnderline})
}

func formatActivity(segment *text.Rich, content *bytes.Buffer, ac discord.Activity) {
	switch ac.Type {
	case discord.GameActivity:
		formatSectionf(segment, content, "Playing %s", ac.Name)
		content.WriteByte('\n')

	case discord.ListeningActivity:
		formatSectionf(segment, content, "Listening to %s", ac.Name)
		content.WriteByte('\n')

	case discord.StreamingActivity:
		formatSectionf(segment, content, "Streaming on %s", ac.Name)
		content.WriteByte('\n')

	case discord.CustomActivity:
		formatSectionf(segment, content, "Status")
		content.WriteByte('\n')

		if ac.Emoji != nil {
			if !ac.Emoji.ID.IsValid() {
				content.WriteString(ac.Emoji.Name)
			} else {
				segmentadd(segment, EmojiSegment{
					Start:    content.Len(),
					Name:     ac.Emoji.Name,
					EmojiURL: ac.Emoji.EmojiURL() + "&size=64",
					Large:    ac.State == "",
				})
			}

			content.WriteByte(' ')
		}

	default:
		formatSectionf(segment, content, "Status")
		content.WriteByte('\n')
	}

	// Insert an image if there's any.
	if ac.Assets != nil && ac.Assets.LargeImage != "" {
		segmentadd(segment, NewLargeActivityImage(content.Len(), ac))
		content.WriteString(" ")
	}

	if ac.Details != "" {
		start, end := writestringbuf(content, ac.Details)
		segmentadd(segment, InlineSegment{start, end, text.AttrBold})
		content.WriteByte('\n')
	}

	if ac.State != "" {
		content.WriteString(ac.State)
	}
}

func getPresence(
	state *ningen.State,
	guildID discord.GuildID, userID discord.UserID) *discord.Activity {

	p, err := state.Presence(guildID, userID)
	if err != nil {
		return nil
	}

	if len(p.Activities) > 0 {
		return &p.Activities[0]
	}

	return p.Game
}

func findRole(roles []discord.Role, id discord.RoleID) (discord.Role, bool) {
	for _, role := range roles {
		if role.ID == id {
			return role, true
		}
	}
	return discord.Role{}, false
}
