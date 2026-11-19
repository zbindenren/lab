package cmd

import (
	"bytes"
	"errors"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	"github.com/ackerr/lab/internal/tokentest"
)

func TestNewAuthLogoutCmd(t *testing.T) {
	const (
		baseURL = "https://gitlab.pnet.ch"
		service = "lab:gitlab.pnet.ch"
		token   = "s3cr3t-token-value"
	)

	keychainDenied := errors.New("exit status 51")

	tests := []struct {
		name        string
		baseURL     string
		stored      map[string]string
		deleteErr   error
		wantOutput  string
		wantError   string
		wantErrorIs error
		wantStored  map[string]string
	}{
		{
			name:       "existing entry is removed",
			baseURL:    baseURL,
			stored:     map[string]string{service: token},
			wantOutput: "removed the keychain entry for gitlab.pnet.ch\n",
		},
		{
			name:       "missing entry is not an error",
			baseURL:    baseURL,
			wantOutput: "no keychain entry for gitlab.pnet.ch\n",
		},
		{
			name:       "host is derived from the configured base url",
			baseURL:    "https://Example.com:8443/gitlab/",
			stored:     map[string]string{"lab:example.com:8443": token},
			wantOutput: "removed the keychain entry for example.com:8443\n",
		},
		{
			name:       "no base url fails like login and removes nothing",
			stored:     map[string]string{service: token, "lab:": token},
			wantError:  "set gitlab.base_url first, use `lab config`",
			wantStored: map[string]string{service: token, "lab:": token},
		},
		{
			name:        "keychain failure is fatal and keeps the entry",
			baseURL:     baseURL,
			stored:      map[string]string{service: token},
			deleteErr:   keychainDenied,
			wantError:   "failed to delete the GitLab token from the OS keychain (" + service + "): " + keychainDenied.Error(),
			wantErrorIs: keychainDenied,
			wantStored:  map[string]string{service: token},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := tokentest.New()
			store.DeleteErr = tt.deleteErr

			for storedService, value := range tt.stored {
				require.NoError(t, store.Set(storedService, value))
			}

			var out bytes.Buffer

			deps := authDeps{
				store: store,
				out:   &out,
			}

			err := logout(deps, tt.baseURL)

			// the whole output is pinned, so token material cannot appear in it
			require.Equal(t, tt.wantOutput, out.String())

			if tt.wantError != "" {
				require.EqualError(t, err, tt.wantError)

				if tt.wantErrorIs != nil {
					require.ErrorIs(t, err, tt.wantErrorIs)
				}
			} else {
				require.NoError(t, err)
			}

			require.Len(t, store.Values, len(tt.wantStored))

			for storedService, value := range tt.wantStored {
				got, getErr := store.Get(storedService)
				require.NoError(t, getErr)
				require.Equal(t, value, got)
			}
		})
	}
}

func TestNewAuthLogoutCmd_RegisteredUnderAuth(t *testing.T) {
	command, _, err := authCmd.Find([]string{"logout"})
	require.NoError(t, err)
	require.Equal(t, "logout", command.Name())
	require.True(t, command.SilenceUsage)
}

func TestNewAuthLogoutCmd_RunsWithoutToken(t *testing.T) {
	tests := []struct {
		name       string
		gitlab     map[string]any
		wantOutput string
		wantError  string
	}{
		{
			name:       "configured base url removes the entry without a token",
			gitlab:     map[string]any{"base_url": "gitlab.example.com/"},
			wantOutput: "no keychain entry for gitlab.example.com\n",
		},
		{
			name:      "missing base url",
			gitlab:    map[string]any{},
			wantError: missingBaseURLMessage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)
			viper.Set("gitlab", tt.gitlab)

			var out bytes.Buffer

			// RunE is called directly: cobra's initializers are not run, so the
			// real config file, the keychain and stdout stay out of the test.
			command := newAuthLogoutCmd(authDeps{
				store: tokentest.New(),
				out:   &out,
			})

			err := command.RunE(command, nil)

			require.Equal(t, tt.wantOutput, out.String())

			if tt.wantError != "" {
				require.EqualError(t, err, tt.wantError)

				return
			}

			require.NoError(t, err)
		})
	}
}
