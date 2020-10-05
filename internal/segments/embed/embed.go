package embed

import (
	"fmt"
	"time"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/state"
	"github.com/diamondburned/cchat-discord/internal/segments/colored"
	"github.com/diamondburned/cchat-discord/internal/segments/inline"
	"github.com/diamondburned/cchat-discord/internal/segments/link"
	"github.com/diamondburned/cchat-discord/internal/segments/renderer"
	"github.com/diamondburned/cchat-discord/internal/urlutils"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/ningen/md"
	"github.com/dustin/go-humanize"
)

var imageExts = []string{".jpg", ".jpeg", ".png", ".webp", ".gif"}

func writeEmbedSep(r *renderer.Text, embedColor discord.Color) {
	if start, end := r.WriteString("---"); embedColor > 0 {
		r.Append(colored.NewSegment(start, end, embedColor.Uint32()))
	}
}

func RenderEmbeds(r *renderer.Text, embeds []discord.Embed, m *discord.Message, s state.Store) {
	for _, embed := range embeds {
		r.StartBlock()
		writeEmbedSep(r, embed.Color)
		r.EnsureBreak()

		RenderEmbed(r, embed, m, s)

		r.EnsureBreak()
		writeEmbedSep(r, embed.Color) // render prepends newline already
		r.EndBlock()
	}
}

func RenderEmbed(r *renderer.Text, embed discord.Embed, m *discord.Message, s state.Store) {
	if a := embed.Author; a != nil && a.Name != "" {
		if a.ProxyIcon != "" {
			r.Append(Author(r.Buffer.Len(), *a))
			r.Buffer.WriteByte(' ')
		}

		start, end := r.WriteString(a.Name)
		r.EnsureBreak()

		if a.URL != "" {
			r.Append(link.NewSegment(start, end, a.URL))
		}
	}

	if embed.Title != "" {
		start, end := r.WriteString(embed.Title)
		r.EnsureBreak()

		// Make the title bold.
		r.Append(inline.NewSegment(start, end, text.AttributeBold))

		if embed.URL != "" {
			r.Append(link.NewSegment(start, end, embed.URL))
		}
	}

	// If we have a thumbnail, then write one.
	if embed.Thumbnail != nil {
		r.Append(Thumbnail(r.Buffer.Len(), *embed.Thumbnail))
		// Guarantee 2 lines because thumbnail needs its own.
		r.StartBlockN(2)
	}

	if embed.Description != "" {
		// Since Discord embeds' descriptions are technically Markdown, we can
		// borrow our Markdown parser for this.
		node := md.ParseWithMessage([]byte(embed.Description), s, m, false)
		// Create a new renderer with inherited state and buffer but a new byte
		// source.
		desc := r.Clone([]byte(embed.Description))
		// Walk using the newly created state.
		desc.Walk(node)
		// Join the created state.
		r.Join(desc)
		// Write a new line.
		r.EnsureBreak()
	}

	if len(embed.Fields) > 0 {
		// Pad two new lines.
		r.StartBlockN(2)

		// Write fields indented once.
		for _, field := range embed.Fields {
			fmt.Fprintf(r.Buffer, "\t%s: %s\n", field.Name, field.Value)
		}
	}

	if f := embed.Footer; f != nil && f.Text != "" {
		if f.ProxyIcon != "" {
			r.Append(Footer(r.Buffer.Len(), *f))
			r.Buffer.WriteByte(' ')
		}

		r.Buffer.WriteString(f.Text)
		r.EnsureBreak()
	}

	if embed.Timestamp.IsValid() {
		if embed.Footer != nil {
			r.Buffer.WriteString(" - ")
		}

		r.Buffer.WriteString(embed.Timestamp.Format(time.RFC1123))
		r.EnsureBreak()
	}

	// Write an image if there's one.
	if embed.Image != nil {
		r.Append(Image(r.Buffer.Len(), *embed.Image))
		// Images take up its own empty line, so we should guarantee 2 empty
		// lines.
		r.StartBlockN(2)
	}
}

func RenderAttachments(r *renderer.Text, attachments []discord.Attachment) {
	// Don't do anything if there are no attachments.
	if len(attachments) == 0 {
		return
	}

	// Start a (small)new block before rendering attachments.
	r.EnsureBreak()

	// Render all attachments. Newline delimited.
	for i, attachment := range attachments {
		RenderAttachment(r, attachment)

		if i != len(attachments) {
			r.Buffer.WriteByte('\n')
		}
	}
}

func RenderAttachment(r *renderer.Text, a discord.Attachment) {
	if urlutils.ExtIs(a.Proxy, imageExts) {
		r.Append(Attachment(r.Buffer.Len(), a))
		return
	}

	start, end := r.WriteStringf(
		"File: %s (%s)",
		a.Filename, humanize.Bytes(a.Size),
	)

	r.Append(link.NewSegment(start, end, a.URL))
}
