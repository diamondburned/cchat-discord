package reference

import (
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/cchat/utils/empty"
)

type MessageID cchat.ID

var _ text.MessageReferencer = (*MessageID)(nil)

func (msgID MessageID) MessageID() string {
	return string(msgID)
}

type MessageSegment struct {
	empty.TextSegment
	start, end int
	messageID  discord.MessageID
}

var _ text.Segment = (*MessageSegment)(nil)

func NewMessageSegment(start, end int, msgID discord.MessageID) MessageSegment {
	return MessageSegment{
		start:     start,
		end:       end,
		messageID: msgID,
	}
}

func (msgseg MessageSegment) Bounds() (start, end int) {
	return msgseg.start, msgseg.end
}

func (msgseg MessageSegment) AsMessageReferencer() text.MessageReferencer {
	return MessageID(msgseg.messageID.String())
}
