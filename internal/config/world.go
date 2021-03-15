package config

import (
	"time"

	"github.com/diamondburned/cchat"
)

var World cchat.Configurator = world

var world = &registry{
	configs: []config{
		{"Mention on Reply", &boolStamp{stamp: 5 * time.Minute, value: false}},
		{"Broadcast Typing", true},
	},
}

// MentionOnReply returns true if message replies should mention users.
func MentionOnReply(timestamp time.Time) bool {
	v := world.get(0).(boolStamp)

	if v.stamp > 0 {
		return timestamp.Add(v.stamp).Before(time.Now())
	}

	return v.value
}

// BroadcastTyping returns true if typing events should be broadcasted.
func BroadcastTyping() bool {
	return world.get(1).(bool)
}
