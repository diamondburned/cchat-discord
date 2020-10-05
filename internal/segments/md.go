package segments

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/state"
	"github.com/diamondburned/cchat-discord/internal/segments/embed"
	"github.com/diamondburned/cchat-discord/internal/segments/renderer"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/ningen/md"
)

func ParseMessage(m *discord.Message, s state.Store) text.Rich {
	var content = []byte(m.Content)
	var node = md.ParseWithMessage(content, s, m, true)

	r := renderer.New(content, node)
	// Register the needed states for some renderers.
	r.WithState(m, s)
	// Render the main message body.
	r.Walk(node)
	// Render the extra bits.
	embed.RenderAttachments(r, m.Attachments)
	embed.RenderEmbeds(r, m.Embeds, m, s)

	return text.Rich{
		Content:  r.String(),
		Segments: r.Segments,
	}
}

func ParseWithMessage(b []byte, m *discord.Message, s state.Store, msg bool) text.Rich {
	node := md.ParseWithMessage(b, s, m, msg)
	return renderer.RenderNode(b, node)
}
