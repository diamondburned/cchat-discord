package mention

import (
	"strings"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/cchat-discord/internal/segments/renderer"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/ningen/v2"
	"github.com/diamondburned/ningen/v2/md"
)

// ChannelName returns the channel name if any, otherwise it formats its own
// name into a list of recipients.
func ChannelName(ch discord.Channel) string {
	switch ch.Type {
	case discord.DirectMessage, discord.GroupDM:
		if len(ch.DMRecipients) > 0 {
			return FormatRecipients(ch.DMRecipients)
		}

	default:
		if ch.Name == "" {
			break
		}

		if ch.NSFW {
			return "#" + ch.Name + " (nsfw)"
		} else {
			return "#" + ch.Name
		}
	}

	return ch.ID.String()
}

// FormatRecipients joins the given list of users into a string listing all
// recipients with English punctuation rules.
func FormatRecipients(users []discord.User) string {
	switch len(users) {
	case 0:
		return ""
	case 1:
		return users[0].Username
	case 2:
		return users[0].Username + " and " + users[1].Username
	}

	var usernames = make([]string, len(users)-1)
	for i, user := range users[:len(users)-1] {
		usernames[i] = user.Username
	}

	return strings.Join(usernames, ", ") + " and " + users[len(users)-1].Username
}

// NewChannelText creates a new rich text describing the given channel fetched
// from the state.
func NewChannelText(s *ningen.State, chID discord.ChannelID) text.Rich {
	ch, err := s.Cabinet.Channel(chID)
	if err != nil {
		return text.Plain(ch.Mention())
	}

	rich := text.Rich{Content: ChannelName(*ch)}
	segment := Segment{
		Start: 0,
		End:   len(rich.Content),
	}

	if ch.Type == discord.DirectMessage && len(ch.DMRecipients) == 1 {
		segment.User = NewUser(ch.DMRecipients[0])
		segment.User.WithState(s)
		segment.User.Prefetch()
	} else {
		segment.Channel = NewChannel(*ch)
	}

	rich.Segments = []text.Segment{segment}
	return rich
}

type Channel struct {
	discord.Channel
}

func NewChannelFromID(s *ningen.State, chID discord.ChannelID) *Channel {
	ch, err := s.Channel(chID)
	if err != nil {
		return &Channel{
			Channel: discord.Channel{ID: chID, Name: "unknown channel"},
		}
	}

	return &Channel{
		Channel: *ch,
	}
}

func NewChannel(ch discord.Channel) *Channel {
	return &Channel{
		Channel: ch,
	}
}

func (ch *Channel) MentionInfo() text.Rich {
	var topic = ch.Topic
	if ch.NSFW {
		topic = "(NSFW)\n" + topic
	}

	if topic == "" {
		return text.Rich{}
	}

	bytes := []byte(topic)

	r := renderer.New(bytes)
	r.Walk(md.Parse(bytes))

	return text.Rich{
		Content:  r.String(),
		Segments: r.Segments,
	}
}
