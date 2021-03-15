package sender

import (
	"github.com/diamondburned/arikawa/v2/api"
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/utils/json/option"
	"github.com/diamondburned/arikawa/v2/utils/sendpart"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/config"
	"github.com/diamondburned/cchat-discord/internal/discord/session/channel/messenger/sender/completer"
	"github.com/diamondburned/cchat-discord/internal/discord/session/channel/shared"
	"github.com/diamondburned/cchat-discord/internal/discord/state"
)

var allowAllMention = []api.AllowedMentionType{
	api.AllowEveryoneMention,
	api.AllowRoleMention,
	api.AllowUserMention,
}

// WrapMessage wraps the given msg to return a new SendMessageData.
func WrapMessage(
	s *state.Instance, ch discord.ChannelID, msg cchat.SendableMessage) api.SendMessageData {

	var send = api.SendMessageData{
		Content: msg.Content(),
	}

	if attacher := msg.AsAttacher(); attacher != nil {
		send.Files = addAttachments(attacher.Attachments())
	}

	if noncer := msg.AsNoncer(); noncer != nil {
		send.Nonce = s.Nonces.Generate(noncer.Nonce())
	}

	if replier := msg.AsReplier(); replier != nil {
		id, err := discord.ParseSnowflake(replier.ReplyingTo())
		if err != nil {
			return send
		}

		send.Reference = &discord.MessageReference{
			MessageID: discord.MessageID(id),
		}
		send.AllowedMentions = &api.AllowedMentions{
			Parse:       allowAllMention,
			RepliedUser: option.False,
		}

		repTo, err := s.Cabinet.Message(ch, discord.MessageID(id))
		if err == nil && config.MentionOnReply(repTo.ID.Time()) {
			send.AllowedMentions.RepliedUser = option.True
		}
	}

	return send
}

type Sender struct {
	shared.Channel
}

var _ cchat.Sender = (*Sender)(nil)

func New(ch shared.Channel) Sender {
	return Sender{ch}
}

func (s Sender) Send(msg cchat.SendableMessage) error {
	_, err := s.State.SendMessageComplex(s.ID, WrapMessage(s.State, s.ID, msg))
	return err
}

// CanAttach returns true if the channel can attach files.
func (s Sender) CanAttach() bool {
	return s.HasPermission(discord.PermissionAttachFiles)
}

func (s Sender) AsCompleter() cchat.Completer {
	return completer.New(s.Channel)
}

func addAttachments(atts []cchat.MessageAttachment) []sendpart.File {
	var files = make([]sendpart.File, len(atts))
	for i, a := range atts {
		files[i] = sendpart.File{
			Name:   a.Name,
			Reader: a,
		}
	}
	return files
}
