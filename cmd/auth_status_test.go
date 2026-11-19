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

func TestAuthStatus(t *testing.T) {
	const (
		baseURL     = "https://gitlab.pnet.ch"
		service     = "lab:gitlab.pnet.ch"
		envToken    = "glpat-from-environment"
		storedToken = "glpat-from-keychain"
		configToken = "glpat-from-config-file"
		// the contract of internal.noTokenMessage, which cmd cannot import
		noToken = "no GitLab token: set GITLAB_TOKEN, run `lab auth login`, or set gitlab.token via `lab config`"
	)

	verifyFailed := errors.New("GET https://gitlab.pnet.ch/api/v4/user: 401 {message: 401 Unauthorized}")
	keychainDenied := errors.New("exit status 51")

	tests := []struct {
		name       string
		baseURL    string
		envToken   string
		stored     map[string]string
		getErr     error
		configTok  string
		verifyErr  error
		wantOutput string
		// wantVerified is the token handed to the verifier, so that the source
		// reported and the token actually used cannot drift apart.
		wantVerified []string
		wantError    string
		wantErrorIs  error
	}{
		{
			name:        "no token in any source",
			baseURL:     baseURL,
			wantOutput:  "host: gitlab.pnet.ch\nsource: none\n",
			wantError:   noToken,
			wantErrorIs: internal.ErrNoToken,
		},
		{
			name:         "environment token",
			baseURL:      baseURL,
			envToken:     envToken,
			wantOutput:   "host: gitlab.pnet.ch\nsource: env\nauthenticated as alice\n",
			wantVerified: []string{envToken},
		},
		{
			name:         "environment outranks an existing keychain entry",
			baseURL:      baseURL,
			envToken:     envToken,
			stored:       map[string]string{service: storedToken},
			wantOutput:   "host: gitlab.pnet.ch\nsource: env\nauthenticated as alice\n",
			wantVerified: []string{envToken},
		},
		{
			name:         "keychain token",
			baseURL:      baseURL,
			stored:       map[string]string{service: storedToken},
			wantOutput:   "host: gitlab.pnet.ch\nsource: keychain\nauthenticated as alice\n",
			wantVerified: []string{storedToken},
		},
		{
			name:         "keychain entry outranks the config file token",
			baseURL:      baseURL,
			stored:       map[string]string{service: storedToken},
			configTok:    configToken,
			wantOutput:   "host: gitlab.pnet.ch\nsource: keychain\nauthenticated as alice\n",
			wantVerified: []string{storedToken},
		},
		{
			name:         "config file token",
			baseURL:      baseURL,
			configTok:    configToken,
			wantOutput:   "host: gitlab.pnet.ch\nsource: config file\nauthenticated as alice\n",
			wantVerified: []string{configToken},
		},
		{
			name:       "host is derived from the configured base url",
			baseURL:    "https://Example.com:8443/gitlab/",
			stored:     map[string]string{"lab:example.com:8443": storedToken},
			wantOutput: "host: example.com:8443\nsource: keychain\nauthenticated as alice\n",
			// the lookup used the derived host, so the token of the normalised
			// service is the one verified
			wantVerified: []string{storedToken},
		},
		{
			name:    "keychain failure is fatal and verifies nothing",
			baseURL: baseURL,
			stored:  map[string]string{service: storedToken},
			getErr:  keychainDenied,
			// a keychain failure is not "no token anywhere", so no source is printed
			wantOutput: "host: gitlab.pnet.ch\n",
			// a locked keychain must not fall back to the config file
			configTok:   configToken,
			wantError:   "failed to read the GitLab token from the OS keychain (" + service + "): " + keychainDenied.Error(),
			wantErrorIs: keychainDenied,
		},
		{
			name:         "verification failure is fatal",
			baseURL:      baseURL,
			stored:       map[string]string{service: storedToken},
			verifyErr:    verifyFailed,
			wantOutput:   "host: gitlab.pnet.ch\nsource: keychain\n",
			wantVerified: []string{storedToken},
			wantError:    "token check failed: " + verifyFailed.Error(),
			wantErrorIs:  verifyFailed,
		},
		{
			name:      "missing base url reports nothing",
			wantError: missingBaseURLMessage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := tokentest.New()
			store.GetErr = tt.getErr

			for storedService, value := range tt.stored {
				require.NoError(t, store.Set(storedService, value))
			}

			var (
				out      bytes.Buffer
				verified []string
			)

			deps := authDeps{
				store: store,
				verify: func(_ context.Context, token string) (string, error) {
					verified = append(verified, token)

					return "alice", tt.verifyErr
				},
				env: func(key string) string {
					require.Equal(t, internal.TokenEnvVar, key)

					return tt.envToken
				},
				read: func() (string, error) {
					t.Fatal("status must never read a token")

					return "", nil
				},
				out: &out,
			}

			err := authStatus(t.Context(), deps, tt.baseURL, tt.configTok)

			require.Equal(t, tt.wantVerified, verified, "verified tokens")
			require.Equal(t, tt.wantOutput, out.String())

			if tt.wantError != "" {
				require.EqualError(t, err, tt.wantError)

				if tt.wantErrorIs != nil {
					require.ErrorIs(t, err, tt.wantErrorIs)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestAuthStatus_NeverPrintsToken(t *testing.T) {
	const (
		baseURL = "https://gitlab.pnet.ch"
		service = "lab:gitlab.pnet.ch"
		token   = "glpat-verysecret-suffix9876"
	)

	// a prefix, a middle fragment and a suffix: none of them may leak either.
	fragments := []string{token, "glpat", "verysecret", "9876"}

	tests := []struct {
		name       string
		envToken   string
		stored     string
		configTok  string
		verifyErr  error
		keychainEr error
	}{
		{name: "success from the environment", envToken: token},
		{name: "success from the keychain", stored: token},
		{name: "success from the config file", configTok: token},
		{name: "verification failure", stored: token, verifyErr: errors.New("401 Unauthorized")},
		{name: "keychain failure", keychainEr: errors.New("exit status 51")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := tokentest.New()
			store.GetErr = tt.keychainEr

			if tt.stored != "" {
				require.NoError(t, store.Set(service, tt.stored))
			}

			var out bytes.Buffer

			deps := authDeps{
				store: store,
				verify: func(context.Context, string) (string, error) {
					return "alice", tt.verifyErr
				},
				env: func(key string) string {
					require.Equal(t, internal.TokenEnvVar, key)

					return tt.envToken
				},
				read: func() (string, error) {
					t.Fatal("status must never read a token")

					return "", nil
				},
				out: &out,
			}

			err := authStatus(t.Context(), deps, baseURL, tt.configTok)

			for _, fragment := range fragments {
				require.NotContains(t, out.String(), fragment)

				if err != nil {
					require.NotContains(t, err.Error(), fragment)
				}
			}
		})
	}
}

func TestNewAuthStatusCmd(t *testing.T) {
	command, _, err := authCmd.Find([]string{"status"})
	require.NoError(t, err)
	require.Equal(t, "status", command.Name())
	require.True(t, command.SilenceUsage, "a token problem must not dump the usage")
	require.NotNil(t, command.RunE, "status reports failure as a returned error")
}

func TestNewAuthStatusCmd_RunsWithoutToken(t *testing.T) {
	const configToken = "glpat-from-config-file"

	tests := []struct {
		name       string
		gitlab     map[string]any
		wantOutput string
		wantError  string
	}{
		{
			name:       "no token and no keychain entry",
			gitlab:     map[string]any{"base_url": "gitlab.example.com/"},
			wantOutput: "host: gitlab.example.com\nsource: none\n",
			wantError:  "no GitLab token: set GITLAB_TOKEN, run `lab auth login`, or set gitlab.token via `lab config`",
		},
		{
			name:       "config file token is reported as its own source",
			gitlab:     map[string]any{"base_url": "gitlab.example.com/", "token": configToken},
			wantOutput: "host: gitlab.example.com\nsource: config file\nauthenticated as alice\n",
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
			command := newAuthStatusCmd(authDeps{
				store: tokentest.New(),
				verify: func(context.Context, string) (string, error) {
					return "alice", nil
				},
				env: func(string) string { return "" },
				read: func() (string, error) {
					t.Fatal("status must never read a token")

					return "", nil
				},
				out: &out,
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
