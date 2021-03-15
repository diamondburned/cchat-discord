package completer

import (
	"sort"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/session/channel/shared"
	"github.com/lithammer/fuzzysearch/fuzzy"
)

type CompleterFunc func(word string) []cchat.CompletionEntry

type ChannelCompleter struct {
	shared.Channel
}

type CompleterPrefixes map[byte]CompleterFunc

type Completer struct {
	Prefixes     CompleterPrefixes
	SlashHandler cchat.Completer
}

const MaxCompletion = 50

func New(ch shared.Channel) cchat.Completer {
	completer := ChannelCompleter{ch}
	return Completer{
		Prefixes: map[byte]CompleterFunc{
			'@': completer.CompleteMentions,
			'#': completer.CompleteChannels,
			':': completer.CompleteEmojis,
		},
	}
}

// CompleteMessage implements message input completion capability for Discord.
// This method supports user mentions, channel mentions and emojis.
//
// For the individual implementations, refer to channel_completion.go.
func (cc Completer) Complete(words []string, i int64) []cchat.CompletionEntry {
	var word = words[i]
	// Word should have at least a character for the char check.
	if len(word) == 0 {
		return nil
	}

	// Always check the first word for slash, not the current word.
	if cc.SlashHandler != nil && words[0][0] == '/' {
		return cc.SlashHandler.Complete(words, i)
	}

	fn, ok := cc.Prefixes[word[0]]
	if !ok {
		return nil
	}

	return fn(word[1:])
}

// rankFunc is the default rank function to use.
func rankFunc(source, target string) int {
	return fuzzy.RankMatchNormalizedFold(source, target)
}

func ensureEntriesMade(entries *[]cchat.CompletionEntry) {
	if *entries == nil {
		*entries = make([]cchat.CompletionEntry, 0, MaxCompletion)
	}
}

func ensureDistancesMade(distances *map[string]int) {
	if *distances == nil {
		*distances = make(map[string]int, MaxCompletion)
	}
}

// sortDistances sorts according to the given Levenshtein distances from the Raw
// string of the entries from most accurate to least accurate.
func sortDistances(entries []cchat.CompletionEntry, distances map[string]int) {
	if len(entries) == 0 {
		return
	}
	// The lower the distance, the more accurate.
	sort.SliceStable(entries, func(i, j int) bool {
		return distances[entries[i].Raw] < distances[entries[j].Raw]
	})
}
