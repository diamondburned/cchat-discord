package segments

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/state"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/ningen/md"
	"github.com/yuin/goldmark/ast"
)

// NameSegment represents a clickable member name; it does not implement colors.
type NameSegment struct {
	start, end int

	guild  discord.Guild
	member discord.Member
}

var (
	_ text.Segment   = (*NameSegment)(nil)
	_ text.Mentioner = (*NameSegment)(nil)
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

func (m NameSegment) Bounds() (start, end int) {
	return m.start, m.end
}

func (m NameSegment) MentionInfo() text.Rich {
	return userInfo(m.guild, m.member)
}

type MentionSegment struct {
	start, end int
	*md.Mention

	store state.Store
	guild discord.Snowflake
}

var (
	_ text.Segment   = (*MentionSegment)(nil)
	_ text.Colorer   = (*MentionSegment)(nil)
	_ text.Mentioner = (*MentionSegment)(nil)
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

func (m MentionSegment) channelInfo() text.Rich {
	content := strings.Builder{}
	content.WriteByte('#')
	content.WriteString(m.Channel.Name)

	if m.Channel.NSFW {
		content.WriteString(" (NSFW)")
	}

	if m.Channel.Topic != "" {
		content.WriteByte('\n')
		content.WriteString(m.Channel.Topic)
	}

	return text.Rich{
		Content: content.String(),
	}
}

func (m MentionSegment) userInfo() text.Rich {
	// // We should have a member if there's nil. Sometimes when the members aren't
	// // prefetched, the markdown parser can miss them. We can check this again.
	// if m.GuildUser.Member == nil && m.guild.Valid() {
	// 	// Best effort; fine if it's nil.
	// 	m.GuildUser.Member, _ = m.store.Member(m.guild, m.GuildUser.ID)
	// }

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

	return userInfo(*g, *m.GuildUser.Member)
}

func userInfo(guild discord.Guild, member discord.Member) text.Rich {
	var content bytes.Buffer
	var segment text.Rich

	// Make a large avatar if there's one.
	if member.User.Avatar != "" {
		segmentadd(&segment, AvatarSegment{
			start: 0,
			url:   member.User.AvatarURL(), // full URL
			text:  "Avatar",
			size:  72, // large
		})
		// Space out.
		content.WriteByte(' ')
	}

	// Write the nickname if there's one; else, write the username only.
	if member.Nick != "" {
		content.WriteString(member.Nick)
		content.WriteByte(' ')

		start, end := writestringbuf(&content, fmt.Sprintf(
			"(%s#%s)",
			member.User.Username,
			member.User.Discriminator,
		))

		segmentadd(&segment, InlineSegment{
			start:      start,
			end:        end,
			attributes: text.AttrDimmed,
		})
	} else {
		content.WriteString(member.User.Username)
		content.WriteByte('#')
		content.WriteString(member.User.Discriminator)
	}

	// Write extra information if any, but only if we have the guild state.
	if len(member.RoleIDs) > 0 && guild.ID.Valid() {
		// Write a prepended new line, as role writes will always prepend a new
		// line. This is to prevent a trailing new line.
		content.WriteString("\n\n--- Roles ---")

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
	}

	// Assign the written content into the text segment and return it.
	segment.Content = content.String()
	return segment
}

func findRole(roles []discord.Role, id discord.Snowflake) (discord.Role, bool) {
	for _, role := range roles {
		if role.ID == id {
			return role, true
		}
	}
	return discord.Role{}, false
}

func (m MentionSegment) roleInfo() text.Rich {
	// We don't have much to write here.
	var segment = text.Rich{
		Content: m.GuildRole.Name,
	}

	// Maybe add a color if we have any.
	if c := m.GuildRole.Color.Uint32(); c > 0 {
		segment.Segments = []text.Segment{
			NewColored(len(m.GuildRole.Name), m.GuildRole.Color.Uint32()),
		}
	}

	return segment
}
