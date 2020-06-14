package discord

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat/text"
)

type Channel struct {
	id      discord.Snowflake
	guildID discord.Snowflake
	name    string
	session *Session
}

func NewChannel(s *Session, ch *discord.Channel) *Channel {
	return &Channel{
		id:      ch.ID,
		guildID: ch.GuildID,
		name:    ch.Name,
		session: s,
	}
}

func (ch *Channel) ID() string {
	return ch.id.String()
}

func (ch *Channel) Name() text.Rich {
	return text.Rich{Content: "#" + ch.name}
}

func (ch *Channel) Nickname(labeler cchat.LabelContainer) error {

}
