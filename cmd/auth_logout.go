package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"

	"github.com/ackerr/lab/internal"
)

// logoutCmd removes the GitLab token stored for the configured host.
var logoutCmd = newAuthLogoutCmd(defaultAuthDeps())

func init() {
	authCmd.AddCommand(logoutCmd)
}

// newAuthLogoutCmd creates the command that removes the GitLab token stored for
// the configured host from the OS keychain.
func newAuthLogoutCmd(deps authDeps) *cobra.Command {
	return &cobra.Command{
		Use:          "logout",
		Short:        "Remove the GitLab token from the OS keychain",
		SilenceUsage: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			// logout needs the GitLab section only: it must work before a token exists.
			internal.SetupGitLabConfig()

			return logout(deps, internal.Config.BaseURL)
		},
	}
}

// logout deletes the keychain entry of the configured host. A missing entry is
// not an error: abandoning a credential is idempotent.
func logout(deps authDeps, baseURL string) error {
	if baseURL == "" {
		return errors.New(missingBaseURLMessage)
	}

	host := internal.GitLabHost(baseURL)
	service := internal.KeychainService(baseURL)

	if err := deps.store.Delete(service); err != nil {
		// only a missing entry is forgiven, so a keychain that refuses the
		// deletion is reported instead of looking like "nothing to remove".
		if errors.Is(err, keyring.ErrNotFound) {
			fmt.Fprintf(deps.out, "no keychain entry for %s\n", host)

			return nil
		}

		return fmt.Errorf("failed to delete the GitLab token from the OS keychain (%s): %w", service, err)
	}

	fmt.Fprintf(deps.out, "removed the keychain entry for %s\n", host)

	return nil
}
