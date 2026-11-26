package internal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestSetupGitLabConfig(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	tests := []struct {
		name          string
		baseURL       string
		codespace     string
		projects      string
		groups        []string
		wantBaseURL   string
		wantCodespace string
		wantProjects  string
		wantGroups    []string
	}{
		{
			name:        "base url without scheme",
			baseURL:     "gitlab.example.com/",
			wantBaseURL: "https://gitlab.example.com",
		},
		{
			name:        "base url with port and subfolder",
			baseURL:     "https://Example.com:8443/gitlab/",
			wantBaseURL: "https://Example.com:8443/gitlab",
		},
		{
			name:          "codespace home expansion",
			baseURL:       "gitlab.example.com",
			codespace:     "~/work/",
			wantBaseURL:   "https://gitlab.example.com",
			wantCodespace: filepath.Join(home, "work"),
		},
		{
			name:         "projects given",
			baseURL:      "gitlab.example.com",
			projects:     "/tmp/projects",
			wantBaseURL:  "https://gitlab.example.com",
			wantProjects: "/tmp/projects",
		},
		{
			name:        "projects default",
			baseURL:     "gitlab.example.com",
			wantBaseURL: "https://gitlab.example.com",
		},
		{
			name:        "groups given",
			baseURL:     "gitlab.example.com",
			groups:      []string{"infrastructure", "linux"},
			wantBaseURL: "https://gitlab.example.com",
			wantGroups:  []string{"infrastructure", "linux"},
		},
		{
			name:        "groups empty falls back to default",
			baseURL:     "gitlab.example.com",
			groups:      []string{},
			wantBaseURL: "https://gitlab.example.com",
			wantGroups:  []string{"linux", "kubernetes"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)

			LabDir = t.TempDir()
			viper.Set("gitlab", map[string]any{
				"base_url":  tt.baseURL,
				"codespace": tt.codespace,
				"projects":  tt.projects,
				"groups":    tt.groups,
			})

			SetupGitLabConfig()

			wantProjects := tt.wantProjects
			if wantProjects == "" {
				wantProjects = filepath.Join(LabDir, ".projects")
			}

			wantGroups := tt.wantGroups
			if wantGroups == nil {
				wantGroups = []string{"linux", "kubernetes"}
			}

			require.Equal(t, tt.wantBaseURL, Config.BaseURL)
			require.Equal(t, tt.wantCodespace, Config.Codespace)
			require.Equal(t, wantProjects, ProjectPath)
			require.Equal(t, wantGroups, Config.Groups)
			require.Empty(t, Config.Token)
		})
	}
}

func TestNormalizeBaseURL(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		want    string
	}{
		{name: "scheme missing", baseURL: "gitlab.example.com", want: "https://gitlab.example.com"},
		{name: "trailing slash", baseURL: "gitlab.example.com/", want: "https://gitlab.example.com"},
		{name: "port and subfolder", baseURL: "https://Example.com:8443/gitlab/", want: "https://Example.com:8443/gitlab"},
		{name: "empty", baseURL: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, normalizeBaseURL(tt.baseURL))
		})
	}
}

func TestSetup_WithoutToken(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	LabDir = t.TempDir()
	viper.Set("main", map[string]any{})
	viper.Set("gitlab", map[string]any{"base_url": "gitlab.example.com"})

	Setup()

	require.Empty(t, Config.Token)
	require.Equal(t, "https://gitlab.example.com", Config.BaseURL)
}
