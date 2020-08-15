package segments

import (
	"fmt"
	"time"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/state"
	"github.com/diamondburned/cchat-discord/urlutils"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/ningen/md"
	"github.com/dustin/go-humanize"
)

var imageExts = []string{".jpg", ".jpeg", ".png", ".webp", ".gif"}

func (r *TextRenderer) writeEmbedSep(embedColor discord.Color) {
	if start, end := r.writeString("---"); embedColor > 0 {
		r.append(NewColoredSegment(start, end, embedColor.Uint32()))
	}
}

func (r *TextRenderer) renderEmbeds(embeds []discord.Embed, m *discord.Message, s state.Store) {
	for _, embed := range embeds {
		r.startBlock()
		r.writeEmbedSep(embed.Color)
		r.ensureBreak()

		r.renderEmbed(embed, m, s)

		r.ensureBreak()
		r.writeEmbedSep(embed.Color) // render prepends newline already
		r.endBlock()
	}
}

func (r *TextRenderer) renderEmbed(embed discord.Embed, m *discord.Message, s state.Store) {
	if a := embed.Author; a != nil && a.Name != "" {
		if a.ProxyIcon != "" {
			r.append(EmbedAuthor(r.buf.Len(), *a))
			r.buf.WriteByte(' ')
		}

		start, end := r.writeString(a.Name)
		r.ensureBreak()

		if a.URL != "" {
			r.append(LinkSegment{
				start,
				end,
				a.URL,
			})
		}
	}

	if embed.Title != "" {
		start, end := r.writeString(embed.Title)
		r.ensureBreak()

		// Make the title bold.
		r.append(InlineSegment{
			start:      start,
			end:        end,
			attributes: text.AttrBold,
		})

		if embed.URL != "" {
			r.append(LinkSegment{
				start,
				end,
				embed.URL,
			})
		}
	}

	// If we have a thumbnail, then write one.
	if embed.Thumbnail != nil {
		r.append(EmbedThumbnail(r.buf.Len(), *embed.Thumbnail))
		// Guarantee 2 lines because thumbnail needs its own.
		r.startBlockN(2)
	}

	if embed.Description != "" {
		// Since Discord embeds' descriptions are technically Markdown, we can
		// borrow our Markdown parser for this.
		node := md.ParseWithMessage([]byte(embed.Description), s, m, false)
		// Create a new renderer with inherited state and buffer but a new byte
		// source.
		desc := r.clone([]byte(embed.Description))
		// Walk using the newly created state.
		desc.walk(node)
		// Join the created state.
		r.join(desc)
		// Write a new line.
		r.ensureBreak()
	}

	if len(embed.Fields) > 0 {
		// Pad two new lines.
		r.startBlockN(2)

		// Write fields indented once.
		for _, field := range embed.Fields {
			fmt.Fprintf(r.buf, "\t%s: %s\n", field.Name, field.Value)
		}
	}

	if f := embed.Footer; f != nil && f.Text != "" {
		if f.ProxyIcon != "" {
			r.append(EmbedFooter(r.buf.Len(), *f))
			r.buf.WriteByte(' ')
		}

		r.buf.WriteString(f.Text)
		r.ensureBreak()
	}

	if embed.Timestamp.IsValid() {
		if embed.Footer != nil {
			r.buf.WriteString(" - ")
		}

		r.buf.WriteString(embed.Timestamp.Format(time.RFC1123))
		r.ensureBreak()
	}

	// Write an image if there's one.
	if embed.Image != nil {
		r.append(EmbedImage(r.buf.Len(), *embed.Image))
		// Images take up its own empty line, so we should guarantee 2 empty
		// lines.
		r.startBlockN(2)
	}
}

func (r *TextRenderer) renderAttachments(attachments []discord.Attachment) {
	// Don't do anything if there are no attachments.
	if len(attachments) == 0 {
		return
	}

	// Start a (small)new block before rendering attachments.
	r.ensureBreak()

	// Render all attachments. Newline delimited.
	for i, attachment := range attachments {
		r.renderAttachment(attachment)

		if i != len(attachments) {
			r.buf.WriteByte('\n')
		}
	}
}

func (r *TextRenderer) renderAttachment(a discord.Attachment) {
	if urlutils.ExtIs(a.Proxy, imageExts) {
		r.append(EmbedAttachment(r.buf.Len(), a))
		return
	}

	start, end := r.writeStringf(
		"File: %s (%s)",
		a.Filename, humanize.Bytes(a.Size),
	)

	r.append(LinkSegment{
		start,
		end,
		a.URL,
	})
}

type AvatarSegment struct {
	start int
	url   string
	text  string
	size  int
}

func EmbedAuthor(start int, a discord.EmbedAuthor) AvatarSegment {
	return AvatarSegment{
		start: start,
		url:   a.ProxyIcon,
		text:  "Avatar",
	}
}

// EmbedFooter uses an avatar segment to comply with Discord.
func EmbedFooter(start int, f discord.EmbedFooter) AvatarSegment {
	return AvatarSegment{
		start: start,
		url:   f.ProxyIcon,
		text:  "Icon",
	}
}

func (a AvatarSegment) Bounds() (int, int) {
	return a.start, a.start
}

// Avatar returns the avatar URL.
func (a AvatarSegment) Avatar() (url string) {
	return a.url
}

// AvatarSize returns the size of a small emoji.
func (a AvatarSegment) AvatarSize() int {
	if a.size > 0 {
		return a.size
	}
	return InlineEmojiSize
}

func (a AvatarSegment) AvatarText() string {
	return a.text
}

type ImageSegment struct {
	start int
	url   string
	w, h  int
	text  string
}

func EmbedImage(start int, i discord.EmbedImage) ImageSegment {
	return ImageSegment{
		start: start,
		url:   i.Proxy,
		w:     int(i.Width),
		h:     int(i.Height),
		text:  fmt.Sprintf("Image (%s)", urlutils.Name(i.URL)),
	}
}

func EmbedThumbnail(start int, t discord.EmbedThumbnail) ImageSegment {
	return ImageSegment{
		start: start,
		url:   t.Proxy,
		w:     int(t.Width),
		h:     int(t.Height),
		text:  fmt.Sprintf("Thumbnail (%s)", urlutils.Name(t.URL)),
	}
}

func EmbedAttachment(start int, a discord.Attachment) ImageSegment {
	return ImageSegment{
		start: start,
		url:   a.Proxy,
		w:     int(a.Width),
		h:     int(a.Height),
		text:  fmt.Sprintf("%s (%s)", a.Filename, humanize.Bytes(a.Size)),
	}
}

func (i ImageSegment) Bounds() (start, end int) {
	return i.start, i.start
}

// Image returns the URL.
func (i ImageSegment) Image() string {
	return i.url
}

func (i ImageSegment) ImageSize() (w, h int) {
	return i.w, i.h
}

func (i ImageSegment) ImageText() string {
	return i.text
}
