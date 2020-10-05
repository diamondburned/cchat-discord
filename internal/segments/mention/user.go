package mention

import (
	"bytes"
	"strings"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/state"
	"github.com/diamondburned/cchat-discord/internal/segments/colored"
	"github.com/diamondburned/cchat-discord/internal/segments/inline"
	"github.com/diamondburned/cchat-discord/internal/segments/segutil"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/cchat/utils/empty"
	"github.com/diamondburned/ningen"
)

// NameSegment represents a clickable member name; it does not implement colors.
type NameSegment struct {
	empty.TextSegment
	start int
	end   int
	um    User
}

var _ text.Segment = (*NameSegment)(nil)

func UserSegment(start, end int, u discord.User) NameSegment {
	return NameSegment{
		start: start,
		end:   end,
		um: User{
			member: discord.Member{User: u},
		},
	}
}

func MemberSegment(start, end int, guild discord.Guild, m discord.Member) NameSegment {
	return NameSegment{
		start: start,
		end:   end,
		um: User{
			guild:  guild,
			member: m,
		},
	}
}

// WithState assigns a ningen state into the given name segment. This allows the
// popovers to have additional information such as user notes.
func (m *NameSegment) WithState(state *ningen.State) {
	m.um.state = state
}

func (m NameSegment) Bounds() (start, end int) {
	return m.start, m.end
}

func (m NameSegment) AsMentioner() text.Mentioner {
	return m.um
}

func (m NameSegment) AsAvatarer() text.Avatarer {
	return m.um
}

type User struct {
	state  state.Store
	guild  discord.Guild
	member discord.Member
}

var (
	_ text.Colorer   = (*User)(nil)
	_ text.Avatarer  = (*User)(nil)
	_ text.Mentioner = (*User)(nil)
)

// NewUser creates a new user mention. If state is of type *ningen.State, then
// it'll fetch additional information asynchronously.
func NewUser(state state.Store, guild discord.GuildID, guser discord.GuildUser) *User {
	if guser.Member == nil {
		m, err := state.Member(guild, guser.ID)
		if err != nil {
			guser.Member = &discord.Member{}
		} else {
			guser.Member = m
		}
	}

	guser.Member.User = guser.User

	// Get the guild for the role slice. If not, then too bad.
	g, err := state.Guild(guild)
	if err != nil {
		g = &discord.Guild{}
	}

	return &User{
		state:  state,
		guild:  *g,
		member: *guser.Member,
	}
}

func (um *User) Color() uint32 {
	g, err := um.state.Guild(um.guild.ID)
	if err != nil {
		return colored.Blurple
	}

	return text.SolidColor(discord.MemberColor(*g, um.member).Uint32())
}

func (um *User) AvatarSize() int {
	return 96
}

func (um *User) AvatarText() string {
	if um.member.Nick != "" {
		return um.member.Nick
	}
	return um.member.User.Username
}

func (um *User) Avatar() (url string) {
	return um.member.User.AvatarURL()
}

func (um *User) MentionInfo() text.Rich {
	var content bytes.Buffer
	var segment text.Rich

	// Write the username if the user has a nickname.
	if um.member.Nick != "" {
		content.WriteString("Username: ")
		content.WriteString(um.member.User.Username)
		content.WriteByte('#')
		content.WriteString(um.member.User.Discriminator)
		content.WriteString("\n\n")
	}

	// Write extra information if any, but only if we have the guild state.
	if len(um.member.RoleIDs) > 0 && um.guild.ID.IsValid() {
		// Write a prepended new line, as role writes will always prepend a new
		// line. This is to prevent a trailing new line.
		formatSectionf(&segment, &content, "Roles")

		for _, id := range um.member.RoleIDs {
			rl, ok := findRole(um.guild.Roles, id)
			if !ok {
				continue
			}

			// Prepend a new line before each item.
			content.WriteByte('\n')
			// Write exactly the role name, then grab the segment and color it.
			start, end := segutil.WriteStringBuf(&content, "@"+rl.Name)
			// But we only add the color if the role has one.
			if rgb := rl.Color.Uint32(); rgb > 0 {
				segutil.Add(&segment, colored.NewSegment(start, end, rgb))
			}
		}

		// End section.
		content.WriteString("\n\n")
	}

	// These information can only be obtained from the state. As such, we check
	// if the state is given.
	if ningenState, ok := um.state.(*ningen.State); ok {
		// Does the user have rich presence? If so, write.
		if p, err := um.state.Presence(um.guild.ID, um.member.User.ID); err == nil {
			for _, ac := range p.Activities {
				formatActivity(&segment, &content, ac)
				content.WriteString("\n\n")
			}
		} else if um.guild.ID.IsValid() {
			// If we're still in a guild, then we can ask Discord for that
			// member with their presence attached.
			ningenState.MemberState.RequestMember(um.guild.ID, um.member.User.ID)
		}

		// Write the user's note if any.
		if note := ningenState.NoteState.Note(um.member.User.ID); note != "" {
			formatSectionf(&segment, &content, "Note")
			content.WriteRune('\n')

			start, end := segutil.WriteStringBuf(&content, note)
			segutil.Add(&segment, inline.NewSegment(start, end, text.AttributeMonospace))

			content.WriteString("\n\n")
		}
	}

	// Assign the written content into the text segment and return it after
	// trimming the trailing new line.
	segment.Content = strings.TrimSuffix(content.String(), "\n")
	return segment
}
