package commands

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/diamondburned/arikawa/bot/extras/arguments"
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/cchat-discord/internal/discord/channel/shared"
	"github.com/pkg/errors"
)

type Commands []Command

// Help renders the help text.
func (cmds Commands) Help() []byte {
	var builder bytes.Buffer
	for _, cmd := range cmds {
		cmd.writeHelp(&builder)
		builder.WriteString("\n")
	}

	return builder.Bytes()
}

// Run runs a command with the given words. It errors out if the command is not
// found.
func (cmds Commands) Run(ch shared.Channel, words []string) ([]byte, error) {
	if words[0] == "help" {
		return cmds.Help(), nil
	}

	cmd := cmds.FindExact(words[0])
	if cmd == nil {
		return nil, fmt.Errorf("unknown command %q, refer to help", words[0])
	}

	return cmd.RunFunc(ch, words[1:])
}

// FindExact finds the exact command. It returns a pointer to the command
// directly in the slice if found. If not, nil is returned.
func (cmds Commands) FindExact(name string) *Command {
	for i, cmd := range cmds {
		if cmd.Name == name {
			return &cmds[i]
		}
	}
	return nil
}

// Find finds commands with the given name. The searching is case insensitive.
func (cmds Commands) Find(name string) []Command {
	name = strings.ToLower(name)

	var found []Command

	for _, cmd := range cmds {
		if strings.HasPrefix(strings.ToLower(cmd.Name), name) {
			// Micro-optimization.
			if found == nil {
				found = make([]Command, 1, len(cmds))
				found[0] = cmd
			} else {
				found = append(found, cmd)
			}
		}
	}

	return found
}

// World is a list of commands.
var World = Commands{
	{
		Name: "send-embed",
		Args: Arguments{"-t title", "-c color", "description"},
		Desc: "Send a basic embed to the current channel",
		RunFunc: func(ch shared.Channel, argv []string) ([]byte, error) {
			var embed discord.Embed
			var color uint // no Uint32Var

			fs := flag.NewFlagSet("send-embed", 0)
			fs.SetOutput(ioutil.Discard)
			fs.StringVar(&embed.Title, "t", "", "Embed title")
			fs.UintVar(&color, "c", 0xFFFFFF, "Embed color")

			if err := fs.Parse(argv); err != nil {
				return nil, err
			}

			embed.Description = fs.Arg(0)
			embed.Color = discord.Color(color)

			m, err := ch.State.SendEmbed(ch.ID, embed)
			if err != nil {
				return nil, errors.Wrap(err, "failed to send embed")
			}

			return bprintf("Message %d sent at %v.", m.ID, m.Timestamp.Time()), nil
		},
	},
	{
		Name: "info",
		Desc: "Print information as JSON",
		RunFunc: func(ch shared.Channel, argv []string) ([]byte, error) {
			channel, err := ch.State.Channel(ch.ID)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get channel")
			}

			b, err := json.MarshalIndent(channel, "", "  ")
			if err != nil {
				return nil, errors.Wrap(err, "failed to marshal to JSON")
			}

			return b, nil
		},
	},
	{
		Name: "list-channels",
		Desc: "Print all channels of this guild and their topics",
		RunFunc: func(ch shared.Channel, argv []string) ([]byte, error) {
			channels, err := ch.State.Channels(ch.GuildID)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get channels")
			}

			var buf bytes.Buffer
			for _, ch := range channels {
				fmt.Fprintf(&buf, "#%s (NSFW %t): %s\n", ch.Name, ch.NSFW, ch.Topic)
			}

			return buf.Bytes(), nil
		},
	},
	{
		Name: "presence",
		Args: Arguments{"mention:user"},
		Desc: "Print JSON of a member/user's presence state",
		RunFunc: func(ch shared.Channel, argv []string) ([]byte, error) {
			if err := assertArgc(argv, 1); err != nil {
				return nil, err
			}

			var user arguments.UserMention
			if err := user.Parse(argv[0]); err != nil {
				return nil, err
			}

			p, err := ch.State.Presence(ch.GuildID, user.ID())
			if err != nil {
				return nil, err
			}

			return renderJSON(p)
		},
	},
	{
		Name: "member",
		Args: Arguments{"mention:user"},
		Desc: "Print JSON of a member/user's member state",
		RunFunc: func(ch shared.Channel, argv []string) ([]byte, error) {
			if err := assertArgc(argv, 1); err != nil {
				return nil, err
			}

			if !ch.GuildID.IsValid() {
				return nil, errors.New("channel not in guild")
			}

			var user arguments.UserMention
			if err := user.Parse(argv[0]); err != nil {
				return nil, err
			}

			m, err := ch.State.Member(ch.GuildID, user.ID())
			if err != nil {
				return nil, err
			}

			return renderJSON(m)
		},
	},
}

func assertArgc(argv []string, argc int) error {
	switch {
	case len(argv) > argc:
		return errors.New("too many arguments")
	case len(argv) < argc:
		return errors.New("too few arguments")
	default:
		return nil
	}
}

func renderJSON(v interface{}) ([]byte, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal to JSON")
	}
	return b, nil
}

// bprintf is sprintf but for byte slices.
func bprintf(f string, v ...interface{}) []byte {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, f, v...)
	return buf.Bytes()
}
