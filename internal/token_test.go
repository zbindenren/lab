package internal

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ackerr/lab/internal/tokentest"
)

// the shared fake is the keychain seam of every auth test
var _ TokenStore = (*tokentest.Store)(nil)

func TestGitLabHost(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		want    string
	}{
		{name: "https scheme", baseURL: "https://gitlab.pnet.ch", want: "gitlab.pnet.ch"},
		{name: "trailing slash, no scheme", baseURL: "gitlab.pnet.ch/", want: "gitlab.pnet.ch"},
		{name: "port, no scheme", baseURL: "gitlab.pnet.ch:8443", want: "gitlab.pnet.ch:8443"},
		{name: "uppercase host, port and subfolder", baseURL: "https://Example.com:8443/gitlab/", want: "example.com:8443"},
		{name: "http scheme and trailing slash", baseURL: "http://gitlab.pnet.ch/", want: "gitlab.pnet.ch"},
		{name: "user info", baseURL: "https://user@gitlab.pnet.ch/", want: "gitlab.pnet.ch"},
		{name: "empty", baseURL: "", want: ""},
		{name: "scheme only", baseURL: "https://", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, GitLabHost(tt.baseURL))
		})
	}
}

func TestKeychainService(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		want    string
	}{
		{name: "host", baseURL: "https://gitlab.pnet.ch", want: "lab:gitlab.pnet.ch"},
		{name: "host with port and subfolder", baseURL: "https://Example.com:8443/gitlab/", want: "lab:example.com:8443"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, KeychainService(tt.baseURL))
		})
	}
}

func TestTokenSource_Values(t *testing.T) {
	require.Equal(t, "env", string(TokenSourceEnv))
	require.Equal(t, "keychain", string(TokenSourceKeychain))
	require.Equal(t, "config file", string(TokenSourceConfigFile))
	require.Equal(t, "none", string(TokenSourceNone))
}

func TestTokenResolver_Resolve(t *testing.T) {
	const (
		baseURL = "https://gitlab.pnet.ch"
		service = "lab:gitlab.pnet.ch"
	)

	keychainDown := errors.New("dbus: couldn't determine address of session bus")

	tests := []struct {
		name        string
		env         string
		keychain    string
		keychainErr error
		configToken string
		wantToken   string
		wantSource  TokenSource
		wantError   string
		wantErrorIs error
		// wantNoErrorIs is asserted to be absent, so "nothing found" cannot be
		// confused with a source that failed.
		wantNoErrorIs error
	}{
		{
			name:        "environment wins over keychain and config file",
			env:         "env-token",
			keychain:    "keychain-token",
			configToken: "config-token",
			wantToken:   "env-token",
			wantSource:  TokenSourceEnv,
		},
		{
			name:        "environment wins over a config file holding the same literal",
			env:         "same-token",
			configToken: "same-token",
			wantToken:   "same-token",
			wantSource:  TokenSourceEnv,
		},
		{
			name:        "keychain wins over the config file",
			keychain:    "keychain-token",
			configToken: "config-token",
			wantToken:   "keychain-token",
			wantSource:  TokenSourceKeychain,
		},
		{
			name:       "keychain only",
			keychain:   "keychain-token",
			wantToken:  "keychain-token",
			wantSource: TokenSourceKeychain,
		},
		{
			name:        "missing keychain entry falls through to the config file",
			configToken: "config-token",
			wantToken:   "config-token",
			wantSource:  TokenSourceConfigFile,
		},
		{
			name:        "no token anywhere",
			wantSource:  TokenSourceNone,
			wantError:   noTokenMessage,
			wantErrorIs: ErrNoToken,
		},
		{
			name:          "keychain failure is fatal even with a config file token",
			keychainErr:   keychainDown,
			configToken:   "config-token",
			wantSource:    TokenSourceNone,
			wantError:     "failed to read the GitLab token from the OS keychain (" + service + "): " + keychainDown.Error(),
			wantErrorIs:   keychainDown,
			wantNoErrorIs: ErrNoToken,
		},
		{
			name:        "environment does not touch a broken keychain",
			env:         "env-token",
			keychainErr: keychainDown,
			wantToken:   "env-token",
			wantSource:  TokenSourceEnv,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := tokentest.New()
			store.GetErr = tt.keychainErr

			if tt.keychain != "" {
				store.Values[service] = tt.keychain
			}

			env := func(key string) string {
				require.Equal(t, TokenEnvVar, key)

				return tt.env
			}

			token, source, err := NewTokenResolver(store, env).Resolve(baseURL, tt.configToken)

			require.Equal(t, tt.wantSource, source)

			if tt.wantError != "" {
				require.EqualError(t, err, tt.wantError)
				require.Empty(t, token)

				if tt.wantErrorIs != nil {
					require.ErrorIs(t, err, tt.wantErrorIs)
				}

				if tt.wantNoErrorIs != nil {
					require.NotErrorIs(t, err, tt.wantNoErrorIs)
				}

				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.wantToken, token)
		})
	}
}

func TestTokenResolver_Resolve_EmptyKeychainEntry(t *testing.T) {
	store := tokentest.New()
	store.Values["lab:gitlab.pnet.ch"] = ""

	token, source, err := NewTokenResolver(store, func(string) string { return "" }).Resolve("https://gitlab.pnet.ch", "config-token")

	require.NoError(t, err)
	require.Equal(t, "config-token", token)
	require.Equal(t, TokenSourceConfigFile, source)
}
