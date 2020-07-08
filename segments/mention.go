package segments

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/cchat-discord/urlutils"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/ningen/md"
	"github.com/yuin/goldmark/ast"
)

const blurple = 0x7289DA

type roleInfo struct {
	name     string
	color    uint32
	position int // used for sorting
}

func (r *TextRenderer) userRoles(user *discord.GuildUser) []roleInfo {
	if user.Member == nil || r.msg == nil || !r.msg.GuildID.Valid() {
		return nil
	}

	var roles = make([]roleInfo, 0, len(user.Member.RoleIDs))

	for _, roleID := range user.Member.RoleIDs {
		r, err := r.store.Role(r.msg.GuildID, roleID)
		if err != nil {
			continue
		}

		roles = append(roles, roleInfo{
			name:     r.Name,
			color:    r.Color.Uint32(), // default 0
			position: r.Position,
		})
	}

	// Sort the roles so the first roles stay in front. We need to do this to
	// both render properly and to get the right role color.
	sort.Slice(roles, func(i, j int) bool {
		return roles[i].position < roles[j].position
	})

	return roles
}

type MentionSegment struct {
	start, end int
	*md.Mention

	// only non-nil if GuildUser is not nil and is in a guild.
	roles []roleInfo
}

var (
	_ text.Segment   = (*MentionSegment)(nil)
	_ text.Colorer   = (*MentionSegment)(nil)
	_ text.Mentioner = (*MentionSegment)(nil)
)

func (r *TextRenderer) mention(n *md.Mention, enter bool) ast.WalkStatus {
	if enter {
		var seg = MentionSegment{Mention: n}

		switch {
		case n.Channel != nil:
			seg.start, seg.end = r.writeString("#" + n.Channel.Name)
		case n.GuildUser != nil:
			seg.start, seg.end = r.writeString("@" + n.GuildUser.Username)
			seg.roles = r.userRoles(n.GuildUser) // get roles as well
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
func (m MentionSegment) Color() uint32 {
	// Try digging through what we have for a color.
	switch {
	case len(m.roles) > 0:
		for _, role := range m.roles {
			if role.color > 0 {
				return role.color
			}
		}
	case m.GuildRole != nil && m.GuildRole.Color > 0:
		return m.GuildRole.Color.Uint32()
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
	var content bytes.Buffer
	var segment text.Rich

	// Make a large avatar if there's one.
	if m.GuildUser != nil {
		segment.Segments = append(segment.Segments, AvatarSegment{
			start: 0,
			url:   urlutils.AvatarURL(m.GuildUser.AvatarURL()),
			text:  "Avatar",
		})
		// Space out.
		content.WriteByte(' ')
	}

	// Write the nickname if there's one; else, write the username only.
	if m.GuildUser.Member != nil && m.GuildUser.Member.Nick != "" {
		content.WriteString(m.GuildUser.Member.Nick)
		content.WriteByte(' ')

		start, end := writestringbuf(&content, fmt.Sprintf(
			"(%s#%s)",
			m.GuildUser.Username,
			m.GuildUser.Discriminator,
		))

		segmentadd(&segment, InlineSegment{
			start:      start,
			end:        end,
			attributes: text.AttrDimmed,
		})
	} else {
		content.WriteString(m.GuildUser.Username)
		content.WriteByte('#')
		content.WriteString(m.GuildUser.Discriminator)
	}

	// Write roles, if any.
	if len(m.roles) > 0 {
		// Write a prepended new line, as role writes will always prepend a new
		// line. This is to prevent a trailing new line.
		content.WriteString("\n---\nRoles")

		for _, role := range m.roles {
			// Prepend a new line before each item.
			content.WriteByte('\n')
			// Write exactly the role name, then grab the segment and color it.
			start, end := writestringbuf(&content, role.name)
			segmentadd(&segment, NewColoredSegment(start, end, role.color))
		}
	}

	// Assign the written content into the text segment and return it.
	segment.Content = content.String()
	return segment
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
