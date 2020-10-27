package authenticate

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/session"
	"github.com/diamondburned/cchat-discord/internal/discord/state"
	"github.com/pkg/errors"
	"github.com/skratchdot/open-golang/open"
)

var ErrDLNotFound = errors.New("DiscordLogin not found. Please install it from the GitHub page.")

type DiscordLoginAuth struct{}

func NewDiscordLogin() cchat.Authenticator {
	return DiscordLoginAuth{}
}

// AuthenticateForm returns an empty slice.
func (DiscordLoginAuth) AuthenticateForm() []cchat.AuthenticateEntry {
	return []cchat.AuthenticateEntry{}
}

// Authenticate pops up discordlogin.
func (DiscordLoginAuth) Authenticate([]string) (cchat.Session, error) {
	path, err := lookPathExtras("discordlogin")
	if err != nil {
		openDiscordLoginPage()
		return nil, ErrDLNotFound
	}

	cmd := &exec.Cmd{Path: path}
	cmd.Stderr = os.Stderr

	// UI will actually block during this time.

	b, err := cmd.Output()
	if err != nil {
		return nil, errors.Wrap(err, "DiscordLogin failed")
	}

	if len(b) == 0 {
		return nil, errors.New("DiscordLogin returned nothing, check Console.")
	}

	i, err := state.NewFromToken(string(b))
	if err != nil {
		return nil, err
	}

	return session.NewFromInstance(i)
}

func openDiscordLoginPage() {
	go open.Run("https://github.com/diamondburned/discordlogin")
}

// lookPathExtras searches for PATH as well as GOBIN and GOPATH/bin.
func lookPathExtras(file string) (string, error) {
	// Add extra PATHs, just in case:
	paths := filepath.SplitList(os.Getenv("PATH"))

	if gobin := os.Getenv("GOBIN"); gobin != "" {
		paths = append(paths, gobin)
	}
	if gopath := os.Getenv("GOPATH"); gopath != "" {
		paths = append(paths, gopath)
	}
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, "go", "bin"))
	}

	const filename = "discordlogin"

	for _, dir := range paths {
		if dir == "" {
			dir = "."
		}

		path := filepath.Join(dir, filename)
		if err := findExecutable(path); err == nil {
			return path, nil
		}
	}

	return "", exec.ErrNotFound
}

func findExecutable(file string) error {
	d, err := os.Stat(file)
	if err != nil {
		return err
	}
	if m := d.Mode(); !m.IsDir() && m&0111 != 0 {
		return nil
	}
	return os.ErrPermission
}
