package segments

import (
	"fmt"
	"time"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/state"
	"github.com/diamondburned/cchat-discord/urlutils"
	"github.com/diamondburned/ningen/md"
	"github.com/dustin/go-humanize"
)

var imageExts = []string{".jpg", ".jpeg", ".png", ".webp", ".gif"}

func (r *TextRenderer) renderEmbeds(embeds []discord.Embed, m *discord.Message, s state.Store) {
	for _, embed := range embeds {
		r.startBlock()
		r.buf.WriteString("---\n")

		r.renderEmbed(embed, m, s)

		r.buf.WriteString("---") // render prepends newline already
		r.endBlock()
	}
}

func (r *TextRenderer) renderEmbed(embed discord.Embed, m *discord.Message, s state.Store) {
	if a := embed.Author; a != nil && a.Name != "" {
		if a.ProxyIcon != "" {
			r.append(EmbedAuthor(r.i(), *a))
			r.buf.WriteByte(' ')
		}

		start, end := r.writeString(a.Name)
		r.buf.WriteByte('\n')

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
		r.buf.WriteByte('\n')

		if embed.URL != "" {
			r.append(LinkSegment{
				start,
				end,
				embed.URL,
			})
		}
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
		r.buf.WriteByte('\n')
	}

	if len(embed.Fields) > 0 {
		// Pad another new line.
		r.buf.WriteByte('\n')

		// Write fields indented once.
		for _, field := range embed.Fields {
			fmt.Fprintf(r.buf, "\t%s: %s\n", field.Name, field.Value)
		}
	}

	if f := embed.Footer; f != nil && f.Text != "" {
		if f.ProxyIcon != "" {
			r.append(EmbedFooter(r.i(), *f))
			r.buf.WriteByte(' ')
		}

		r.buf.WriteString(f.Text)
		r.buf.WriteByte('\n')
	}

	if embed.Timestamp.Valid() {
		if embed.Footer != nil {
			r.buf.WriteString(" - ")
		}

		r.buf.WriteString(embed.Timestamp.Format(time.RFC1123))
		r.buf.WriteByte('\n')
	}
}

func (r *TextRenderer) renderAttachments(attachments []discord.Attachment) {
	// Don't do anything if there are no attachments.
	if len(attachments) == 0 {
		return
	}

	// Start a new block before rendering attachments.
	r.startBlock()

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
		r.append(EmbedAttachment(r.i(), a))
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

func EmbedImage(start int, i discord.EmbedImage, text string) ImageSegment {
	return ImageSegment{
		start: start,
		url:   i.Proxy,
		w:     int(i.Width),
		h:     int(i.Height),
		text:  text,
	}
}

func EmbedThumbnail(start int, t discord.EmbedThumbnail, text string) ImageSegment {
	return ImageSegment{
		start: start,
		url:   t.Proxy,
		w:     int(t.Width),
		h:     int(t.Height),
		text:  text,
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