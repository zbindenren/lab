package internal

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewClient_Token(t *testing.T) {
	tests := []struct {
		name        string
		baseURL     string
		token       string
		wantURL     string
		wantError   string
		wantNoToken bool
	}{
		{
			name:    "explicit token",
			baseURL: "https://gitlab.example.com",
			token:   "glpat-secret",
			wantURL: "https://gitlab.example.com/api/v4/",
		},
		{
			name:        "no token",
			baseURL:     "https://gitlab.example.com",
			wantError:   "no GitLab token: set GITLAB_TOKEN, run `lab auth login`, or set gitlab.token via `lab config`",
			wantNoToken: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := newClient(tt.baseURL, tt.token)
			if tt.wantError != "" {
				require.EqualError(t, err, tt.wantError)
				require.Equal(t, tt.wantNoToken, errors.Is(err, ErrNoToken))
				require.Nil(t, client)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.wantURL, client.BaseURL().String())
		})
	}
}
