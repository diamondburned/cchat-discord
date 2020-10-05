package mention

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/cchat-discord/internal/segments/colored"
	"github.com/diamondburned/cchat/text"
)

type Role struct {
	discord.Role
}

func NewRole(role discord.Role) *Role {
	return &Role{role}
}

func (r *Role) Color() uint32 {
	if r.Role.Color == 0 {
		return colored.Blurple
	}
	return text.SolidColor(r.Role.Color.Uint32())
}
