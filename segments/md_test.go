package segments

import (
	"errors"
	"log"
	"testing"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/state"
	"github.com/diamondburned/cchat/text"
	"github.com/go-test/deep"
)

type segtest struct {
	in  string
	out text.Rich
}

func mksegtest(in string, out string, segs ...text.Segment) segtest {
	return segtest{
		in:  in,
		out: text.Rich{Content: out, Segments: segs},
	}
}

func init() {
	deep.CompareUnexportedFields = true
}

func TestParse(t *testing.T) {
	var tests = []segtest{
		mksegtest(
			"This makes me <:Thonk:456835728559702052>",
			"This makes me ",
			EmojiSegment{
				start:    14,
				large:    false,
				name:     "Thonk",
				emojiURL: "https://cdn.discordapp.com/emojis/456835728559702052.png?v=1&size=64",
			},
		),
		mksegtest(
			"This is https://google.com",
			"This is https://google.com",
			LinkSegment{8, 26, "https://google.com"},
		),
		mksegtest(
			"**bold and *italics*** text",
			"bold and italics text",
			InlineSegment{0, 9, text.AttrBold},
			InlineSegment{9, 16, text.AttrBold | text.AttrItalics},
		),
		mksegtest(
			"> imagine best trap\n> not being astolfo",
			"> imagine best trap\n> not being astolfo",
			BlockquoteSegment{0, 39},
		),
		mksegtest(
			"```go\npackage main\n\nfunc main() {}```",
			"package main\n\nfunc main() {}",
			CodeblockSegment{0, 28, "go"},
		),
	}

	for _, test := range tests {
		text := Parse([]byte(test.in))
		log.Printf("Output: %#v\n", text)

		assert(t, text, test)
	}
}

func TestMessage(t *testing.T) {
	var msg = discord.Message{
		ID:      69420,
		Content: "<@1> where's <#2>",
		Mentions: []discord.GuildUser{{
			User: discord.User{
				ID:       1,
				Username: "astolfo",
			},
		}},
	}

	var store = mockStore{}

	text := ParseMessage(&msg, store)
	log.Printf("Output: %#v\n", text)

	assert(t, text, mksegtest(
		"Message",
		"@astolfo where's #traps",
		MentionSegment{0, 8},
		MentionSegment{17, 23},
	))
}

type mockStore struct {
	state.NoopStore
}

func (mockStore) Channel(id discord.Snowflake) (*discord.Channel, error) {
	if id != 2 {
		return nil, errors.New("Unknown channel")
	}

	return &discord.Channel{
		ID:   2,
		Name: "traps",
	}, nil
}

func assert(t *testing.T, got text.Rich, expect segtest) {
	t.Helper()

	if diff := deep.Equal(got, expect.out); diff != nil {
		t.Logf("Got %d error(s) for %q", len(diff), expect.in)

		for _, d := range diff {
			t.Error("(got != expected) " + d)
		}
	}
}
