package mention

import (
	"bytes"
	"fmt"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/cchat-discord/internal/segments/emoji"
	"github.com/diamondburned/cchat-discord/internal/segments/inline"
	"github.com/diamondburned/cchat-discord/internal/segments/segutil"
	"github.com/diamondburned/cchat-discord/internal/urlutils"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/cchat/utils/empty"
	"github.com/diamondburned/ningen"
)

type LargeActivityImage struct {
	empty.TextSegment
	start int
	url   string
	text  string
}

var (
	_ text.Imager  = (*LargeActivityImage)(nil)
	_ text.Segment = (*LargeActivityImage)(nil)
)

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
func (i LargeActivityImage) AsImager() text.Imager    { return i }
func (i LargeActivityImage) Image() string            { return i.url }
func (i LargeActivityImage) ImageSize() (w, h int)    { return 60, 60 }
func (i LargeActivityImage) ImageText() string        { return i.text }

func formatSectionf(segment *text.Rich, content *bytes.Buffer, f string, argv ...interface{}) {
	// Treat f as a regular string at first.
	var str = fmt.Sprintf("%s", f)

	// If there are argvs, then treat f as a format string.
	if len(argv) > 0 {
		str = fmt.Sprintf(str, argv...)
	}

	start, end := segutil.WriteStringBuf(content, str)
	segutil.Add(segment, inline.NewSegment(
		start, end,
		text.AttributeBold,
		text.AttributeUnderline,
	))
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
				segutil.Add(segment, emoji.Segment{
					Start: content.Len(),
					Emoji: emoji.EmojiFromDiscord(*ac.Emoji, ac.State == ""),
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
		segutil.Add(segment, NewLargeActivityImage(content.Len(), ac))
		content.WriteString(" ")
	}

	if ac.Details != "" {
		start, end := segutil.WriteStringBuf(content, ac.Details)
		segutil.Add(segment, inline.NewSegment(start, end, text.AttributeBold))
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
