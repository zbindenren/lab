package cmd

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	"github.com/ackerr/lab/internal"
	"github.com/ackerr/lab/internal/tokentest"
)

func TestLogin(t *testing.T) {
	const (
		baseURL = "https://gitlab.example.com"
		service = "lab:gitlab.example.com"
		pat     = "glpat-secret"
		envNote = "GITLAB_TOKEN is set and takes precedence over the token just stored"
	)

	verifyFailed := errors.New("GET https://gitlab.example.com/api/v4/user: 401 {message: 401 Unauthorized}")
	storeFailed := errors.New("exit status 51")
	readFailed := errors.New("inappropriate ioctl for device")

	tests := []struct {
		name          string
		baseURL       string
		input         string
		readErr       error
		username      string
		verifyErr     error
		setErr        error
		envToken      string
		wantReads     int
		wantVerifies  int
		wantStored    map[string]string
		wantOutput    []string
		wantUnprinted []string
		wantError     string
	}{
		{
			name:          "stores a verified token and prints the username",
			baseURL:       baseURL,
			input:         "  " + pat + "\n",
			username:      "alice",
			wantReads:     1,
			wantVerifies:  1,
			wantStored:    map[string]string{service: pat},
			wantOutput:    []string{"authenticated as alice"},
			wantUnprinted: []string{"GITLAB_TOKEN is set"},
		},
		{
			name:         "environment note when GITLAB_TOKEN is set",
			baseURL:      baseURL,
			input:        pat + "\n",
			username:     "alice",
			envToken:     "glpat-env",
			wantReads:    1,
			wantVerifies: 1,
			wantStored:   map[string]string{service: pat},
			wantOutput:   []string{"authenticated as alice", envNote},
		},
		{
			name:         "verification failure stores nothing",
			baseURL:      baseURL,
			input:        pat + "\n",
			verifyErr:    verifyFailed,
			wantReads:    1,
			wantVerifies: 1,
			wantError:    "token check failed: " + verifyFailed.Error(),
		},
		{
			name:      "empty input verifies and stores nothing",
			baseURL:   baseURL,
			input:     " \n\t",
			wantReads: 1,
			wantError: "no token provided on stdin",
		},
		{
			name:      "missing base url stores nothing",
			input:     pat + "\n",
			wantError: "set gitlab.base_url first, use `lab config`",
		},
		{
			name:         "store failure is a hard error naming the service",
			baseURL:      baseURL,
			input:        pat + "\n",
			username:     "alice",
			setErr:       storeFailed,
			wantReads:    1,
			wantVerifies: 1,
			wantError:    "failed to store the GitLab token in the OS keychain (" + service + "): " + storeFailed.Error(),
		},
		{
			name:      "read failure stores nothing",
			baseURL:   baseURL,
			readErr:   readFailed,
			wantReads: 1,
			wantError: "failed to read the token: " + readFailed.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := tokentest.New()
			store.SetErr = tt.setErr

			var (
				out             bytes.Buffer
				reads, verifies int
			)

			deps := authDeps{
				store: store,
				verify: func(context.Context, string) (string, error) {
					verifies++

					return tt.username, tt.verifyErr
				},
				env: func(key string) string {
					require.Equal(t, internal.TokenEnvVar, key)

					return tt.envToken
				},
				read: func() (string, error) {
					reads++

					return tt.input, tt.readErr
				},
				out: &out,
			}

			err := login(t.Context(), deps, tt.baseURL)

			require.Equal(t, tt.wantReads, reads, "token reads")
			require.Equal(t, tt.wantVerifies, verifies, "verifications")

			if tt.wantStored == nil {
				require.Empty(t, store.Values, "nothing stored")
			} else {
				require.Equal(t, tt.wantStored, store.Values)
			}

			if tt.wantError != "" {
				require.EqualError(t, err, tt.wantError)
				require.Empty(t, out.String(), "nothing printed on failure")

				return
			}

			require.NoError(t, err)

			for _, want := range tt.wantOutput {
				require.Contains(t, out.String(), want)
			}

			for _, unwanted := range tt.wantUnprinted {
				require.NotContains(t, out.String(), unwanted)
			}
		})
	}
}

func TestLogin_NeverPrintsToken(t *testing.T) {
	const (
		baseURL = "https://gitlab.example.com"
		pat     = "glpat-must-not-appear-anywhere"
	)

	tests := []struct {
		name      string
		envToken  string
		verifyErr error
	}{
		{name: "success with environment note", envToken: "glpat-env"},
		{name: "verification failure", verifyErr: errors.New("401 Unauthorized")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer

			deps := authDeps{
				store: tokentest.New(),
				verify: func(context.Context, string) (string, error) {
					return "alice", tt.verifyErr
				},
				env: func(string) string {
					return tt.envToken
				},
				read: func() (string, error) {
					return pat + "\n", nil
				},
				out: &out,
			}

			err := login(t.Context(), deps, baseURL)

			require.NotContains(t, out.String(), pat)
			require.NotContains(t, out.String(), "must-not-appear")

			if err != nil {
				require.NotContains(t, err.Error(), pat)
				require.NotContains(t, err.Error(), "must-not-appear")
			}
		})
	}
}

func TestNewLoginCmd(t *testing.T) {
	require.Contains(t, authCmd.Commands(), loginCmd, "login is registered under lab auth")
	require.Equal(t, "login", loginCmd.Name())
	require.True(t, loginCmd.SilenceUsage, "a login failure must not dump the usage")
	require.NotNil(t, loginCmd.RunE, "login reports failure as a returned error")

	newDeps := func(out *bytes.Buffer, store *tokentest.Store) authDeps {
		return authDeps{
			store: store,
			verify: func(context.Context, string) (string, error) {
				return "alice", nil
			},
			env: func(string) string {
				return ""
			},
			read: func() (string, error) {
				return "glpat-secret\n", nil
			},
			out: out,
		}
	}

	tests := []struct {
		name       string
		gitlab     map[string]any
		wantError  string
		wantStored map[string]string
	}{
		{
			name:       "requires no token to already exist",
			gitlab:     map[string]any{"base_url": "gitlab.example.com/"},
			wantStored: map[string]string{"lab:gitlab.example.com": "glpat-secret"},
		},
		{
			name:      "missing base url",
			gitlab:    map[string]any{},
			wantError: "set gitlab.base_url first, use `lab config`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)
			viper.Set("gitlab", tt.gitlab)

			store := tokentest.New()

			var out bytes.Buffer

			// RunE is called directly: cobra's initializers are not run, so the real
			// config file, the keychain and stdout stay out of the test.
			cmd := newLoginCmd(newDeps(&out, store))

			err := cmd.RunE(cmd, nil)

			if tt.wantError != "" {
				require.EqualError(t, err, tt.wantError)
				require.Empty(t, out.String())
				require.Empty(t, store.Values)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.wantStored, store.Values)
			require.Contains(t, out.String(), "authenticated as alice")
		})
	}
}
