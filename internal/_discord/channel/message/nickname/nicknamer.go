package nickname

import (
	"context"
	"fmt"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/channel/shared"
	"github.com/diamondburned/cchat-discord/internal/funcutil"
	"github.com/diamondburned/cchat-discord/internal/segments/colored"
	"github.com/diamondburned/cchat/text"
)

type Nicknamer struct {
	userID discord.UserID
	shared.Channel
}

func New(ch shared.Channel) cchat.Nicknamer {
	return NewMember(ch.State.UserID, ch)
}

func NewMember(userID discord.UserID, ch shared.Channel) cchat.Nicknamer {
	return Nicknamer{userID, ch}
}

func (nn Nicknamer) Nickname(ctx context.Context, labeler cchat.LabelContainer) (func(), error) {
	// We don't have a nickname if we're not in a guild.
	if !nn.GuildID.IsValid() {
		// Use the current user.
		u, err := nn.State.Cabinet.Me()
		if err == nil {
			labeler.SetLabel(text.Plain(fmt.Sprintf("%s#%s", u.Username, u.Discriminator)))
		}

		return func() {}, nil
	}

	nn.tryNicknameLabel(ctx, labeler)

	return funcutil.JoinCancels(
		nn.State.AddHandler(func(chunks *gateway.GuildMembersChunkEvent) {
			if chunks.GuildID != nn.GuildID {
				return
			}
			for _, member := range chunks.Members {
				if member.User.ID == nn.userID {
					nn.setMember(labeler, member)
					break
				}
			}
		}),
		nn.State.AddHandler(func(g *gateway.GuildMemberUpdateEvent) {
			if g.GuildID == nn.GuildID && g.User.ID == nn.userID {
				nn.setMember(labeler, discord.Member{
					User:    g.User,
					Nick:    g.Nick,
					RoleIDs: g.RoleIDs,
				})
			}
		}),
	), nil
}

func (nn Nicknamer) tryNicknameLabel(ctx context.Context, labeler cchat.LabelContainer) {
	state := nn.State.WithContext(ctx)

	m, err := state.Cabinet.Member(nn.GuildID, nn.userID)
	if err == nil {
		nn.setMember(labeler, *m)
	}
}

func (nn Nicknamer) setMember(labeler cchat.LabelContainer, m discord.Member) {
	var rich = text.Rich{Content: m.User.Username}
	if m.Nick != "" {
		rich.Content = m.Nick
	}

	guild, err := nn.State.Cabinet.Guild(nn.GuildID)
	if err == nil {
		if color := discord.MemberColor(*guild, m); color > 0 {
			rich.Segments = []text.Segment{
				colored.New(len(rich.Content), color.Uint32()),
			}
		}
	}

	labeler.SetLabel(rich)
}
