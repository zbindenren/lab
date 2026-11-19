package cmd

import (
	"context"
	"io"
	"os"

	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/ackerr/lab/internal"
)

// missingBaseURLMessage is the failure the auth commands share when the config
// names no GitLab instance, so none of them can act.
const missingBaseURLMessage = "set gitlab.base_url first, use `lab config`"

// authDeps is the dependency bundle the auth commands are constructed with, so
// they can be exercised without a keychain, a network or a TTY.
type authDeps struct {
	// store is the OS keychain boundary.
	store internal.TokenStore
	// verify checks a candidate token against the API and returns its username.
	verify func(ctx context.Context, token string) (username string, err error)
	// env is the environment lookup, which must read GITLAB_TOKEN directly.
	env func(string) string
	// read returns the candidate token: a hidden prompt on a TTY, stdin when piped.
	read func() (string, error)
	// out is where the commands report their result.
	out io.Writer
}

// authCmd groups the commands that manage the GitLab token in the OS keychain.
// The subcommands live in their own files and register themselves in init().
var authCmd = &cobra.Command{
	Use:          "auth",
	Short:        "Manage the GitLab token stored in the OS keychain",
	SilenceUsage: true,
}

func init() {
	rootCmd.AddCommand(authCmd)
}

// defaultAuthDeps returns the dependency bundle of the production auth commands.
func defaultAuthDeps() authDeps {
	return authDeps{
		store:  internal.NewKeychainStore(),
		verify: verifyToken,
		env:    os.Getenv,
		read:   readTokenInput,
		out:    os.Stdout,
	}
}

// verifyToken checks a candidate token against the API of the configured GitLab
// instance and returns the username it belongs to. The API error is returned
// unchanged, so callers can report why the check failed.
func verifyToken(ctx context.Context, token string) (string, error) {
	user, _, err := internal.NewClientWithToken(token).Users.CurrentUser(gitlab.WithContext(ctx))
	if err != nil {
		return "", err
	}

	return user.Username, nil
}
