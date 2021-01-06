package reference

import (
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/segments/segutil"
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

// Write appends to the given rich text the reference to the message ID with the
// given text.
func Write(rich *text.Rich, msgID discord.MessageID, text string) {
	start, end := segutil.Write(rich, text)
	segutil.Add(rich, NewMessageSegment(start, end, msgID))
}

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
