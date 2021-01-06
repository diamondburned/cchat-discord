package mention

import (
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/cchat-discord/internal/segments/renderer"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/ningen/v2"
	"github.com/diamondburned/ningen/v2/md"
)

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
