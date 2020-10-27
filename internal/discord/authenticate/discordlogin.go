package authenticate

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-discord/internal/discord/session"
	"github.com/diamondburned/cchat-discord/internal/discord/state"
	"github.com/diamondburned/cchat/text"
	"github.com/pkg/errors"
	"github.com/skratchdot/open-golang/open"
)

var ErrDLNotFound = errors.New("DiscordLogin not found. Please install it from the GitHub page.")

// DiscordLoginAuth is a first stage authenticator that allows the user to
// authenticate using DiscordLogin. The Authenticate() function will exec up the
// application if possible. If not, it'll try and exec up a browser.
type DiscordLoginAuth struct{}

func NewDiscordLogin() DiscordLoginAuth {
	return DiscordLoginAuth{}
}

func (DiscordLoginAuth) Name() text.Rich {
	return text.Plain("DiscordLogin")
}

func (DiscordLoginAuth) Description() text.Rich {
	return text.Plain("Log in using DiscordLogin, a WebKit application.")
}

// AuthenticateForm returns an empty slice.
func (DiscordLoginAuth) AuthenticateForm() []cchat.AuthenticateEntry {
	return []cchat.AuthenticateEntry{}
}

// Authenticate pops up DiscordLogin.
func (DiscordLoginAuth) Authenticate([]string) (cchat.Session, cchat.AuthenticateError) {
	path, err := lookPathExtras("discordlogin")
	if err != nil {
		openDiscordLoginPage()
		return nil, cchat.WrapAuthenticateError(ErrDLNotFound)
	}

	cmd := &exec.Cmd{Path: path}
	cmd.Stderr = os.Stderr

	// UI will actually block during this time.

	b, err := cmd.Output()
	if err != nil {
		return nil, cchat.WrapAuthenticateError(errors.Wrap(err, "DiscordLogin failed"))
	}

	if len(b) == 0 {
		return nil, cchat.WrapAuthenticateError(
			errors.New("DiscordLogin returned nothing, check Console."),
		)
	}

	i, err := state.NewFromToken(string(b))
	if err != nil {
		return nil, cchat.WrapAuthenticateError(errors.Wrap(err, "failed to use token"))
	}

	s, err := session.NewFromInstance(i)
	if err != nil {
		return nil, cchat.WrapAuthenticateError(errors.Wrap(err, "failed to make a session"))
	}

	return s, nil
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
