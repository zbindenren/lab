package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/ackerr/lab/internal"
)

// loginCmd verifies a personal access token and stores it in the OS keychain.
var loginCmd = newLoginCmd(defaultAuthDeps())

func init() {
	authCmd.AddCommand(loginCmd)
}

// newLoginCmd creates the login command from the auth dependency bundle.
func newLoginCmd(deps authDeps) *cobra.Command {
	return &cobra.Command{
		Use:          "login",
		Short:        "Verify a GitLab token and store it in the OS keychain",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// login needs the GitLab section only: it must run before a token exists.
			internal.SetupGitLabConfig()

			return login(cmd.Context(), deps, internal.Config.BaseURL)
		},
	}
}

// login verifies the candidate token and stores it under lab:<host>. Nothing is
// written to the store before the API accepts the token, so a typo or an expired
// token never replaces a working entry.
func login(ctx context.Context, deps authDeps, baseURL string) error {
	if baseURL == "" {
		return errors.New(missingBaseURLMessage)
	}

	token, err := readToken(deps.read)
	if err != nil {
		return err
	}

	username, err := deps.verify(ctx, token)
	if err != nil {
		return fmt.Errorf("token check failed: %w", err)
	}

	service := internal.KeychainService(baseURL)

	if err := deps.store.Set(service, token); err != nil {
		return fmt.Errorf("failed to store the GitLab token in the OS keychain (%s): %w", service, err)
	}

	fmt.Fprintf(deps.out, "authenticated as %s\n", username)

	if deps.env(internal.TokenEnvVar) != "" {
		fmt.Fprintf(deps.out, "%s is set and takes precedence over the token just stored\n", internal.TokenEnvVar)
	}

	return nil
}

// readToken returns the trimmed candidate token and rejects empty input, so an
// empty prompt or an empty pipe neither reaches the API nor the keychain.
func readToken(read func() (string, error)) (string, error) {
	input, err := read()
	if err != nil {
		return "", fmt.Errorf("failed to read the token: %w", err)
	}

	token := strings.TrimSpace(input)
	if token == "" {
		return "", errors.New("no token provided on stdin")
	}

	return token, nil
}

// readTokenInput reads the token from a hidden prompt when stdin is a terminal
// and from stdin itself when it is piped.
func readTokenInput() (string, error) {
	if term.IsTerminal(int(os.Stdin.Fd())) {
		return promptToken()
	}

	buf, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", err
	}

	return string(buf), nil
}

// promptToken writes the prompt to stderr and reads the token without echo, so
// it never reaches the screen or the terminal scrollback.
func promptToken() (string, error) {
	fmt.Fprint(os.Stderr, "Token: ")

	token, err := term.ReadPassword(int(os.Stdin.Fd()))

	// the user's Enter is not echoed, so the line must be closed by lab.
	fmt.Fprintln(os.Stderr)

	if err != nil {
		return "", err
	}

	return string(token), nil
}
