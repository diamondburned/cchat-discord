package mention

import (
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/cchat-discord/internal/segments/renderer"
	"github.com/diamondburned/cchat/text"
)

type Channel struct {
	discord.Channel
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

	return renderer.Parse([]byte(topic))
}
