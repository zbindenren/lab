package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ackerr/lab/internal"
)

// statusCmd reports where the GitLab token lab would use comes from and whether
// the API accepts it.
var statusCmd = newAuthStatusCmd(defaultAuthDeps())

func init() {
	authCmd.AddCommand(statusCmd)
}

// newAuthStatusCmd creates the status command from the auth dependency bundle.
func newAuthStatusCmd(deps authDeps) *cobra.Command {
	return &cobra.Command{
		Use:          "status",
		Short:        "Show which token source is active and whether GitLab accepts the token",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// status must be able to report a missing token, so it loads the
			// GitLab section only: Setup() exits when no token is configured.
			internal.SetupGitLabConfig()

			return authStatus(cmd.Context(), deps, internal.Config.BaseURL, internal.Config.Token)
		},
	}
}

// authStatus prints the host lab talks to, the source the active token came from
// and the username that token belongs to. It returns an error when no token can
// be resolved or the API rejects it, which turns into exit code 1.
// Only the source name and the host leave the resolver: the token itself is never
// printed, so the output stays safe to paste into a ticket.
func authStatus(ctx context.Context, deps authDeps, baseURL, configToken string) error {
	if baseURL == "" {
		return errors.New(missingBaseURLMessage)
	}

	fmt.Fprintf(deps.out, "host: %s\n", internal.GitLabHost(baseURL))

	// the resolver is the ladder itself: reporting the source by any other means
	// would be a second implementation of the same order.
	token, source, err := internal.NewTokenResolver(deps.store, deps.env).Resolve(baseURL, configToken)

	// ErrNoToken is the answer "no token anywhere", which status exists to give.
	// Any other error is a failed source, not a missing token, and reporting it as
	// source: none would blame the wrong thing.
	if err != nil && !errors.Is(err, internal.ErrNoToken) {
		return err
	}

	fmt.Fprintf(deps.out, "source: %s\n", source)

	if err != nil {
		return err
	}

	username, err := deps.verify(ctx, token)
	if err != nil {
		return fmt.Errorf("token check failed: %w", err)
	}

	fmt.Fprintf(deps.out, "authenticated as %s\n", username)

	return nil
}
