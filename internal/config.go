// Package internal provides lab's configuration, git and GitLab helpers.
package internal

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"

	"github.com/a8m/envsubst"
	"github.com/spf13/viper"

	"github.com/ackerr/lab/utils"
)

const content = `[gitlab]
# gitlab domain, like https://gitlab.com
base_url = "$GITLAB_BASE_URL"

# gitlab access token
# Last resort: GITLAB_TOKEN and the token stored by lab auth login win over this entry.
# Leave it empty to read the token from the keychain.
token = "$GITLAB_TOKEN"

# gitlab projects file
# default $HOME/config/.lab/.projects
projects = ""

# If set, lab clone and lab cs will use this path as target path
# default empty
codespace = ""

# If set, lab clone will auto set user.name in repo gitconfig
# default empty
name = ""

# If set, lab clone will auto set user.email in repo gitconfig
# default empty
email = ""

# Groups whose projects lab sync collects, including all their subgroups.
# default ["linux", "kubernetes"]
groups = []

[main]
# If set 1, it will use fzf as fuzzy finder, default use go-fuzzyfinder
# default 0
fzf = 0

# lab clone extra custom git clone config
# example clone_opts="--origin ackerr --branch fix"
# default empty
clone_opts = ""
`

// syncGroups is the group list used when the config file names none.
// It returns a fresh slice so Config.Groups never aliases shared state.
func syncGroups() []string {
	return []string{"linux", "kubernetes"}
}

var (
	// Config is the gitlab section of the lab config file.
	Config *gitlabConfig
	// MainConfig is the main section of the lab config file.
	MainConfig *mainConfig
	// LabDir is the directory holding the lab config file.
	LabDir string
	// ConfigPath is the path of the lab config file.
	ConfigPath string
	// ProjectPath is the file the synced project paths are stored in.
	ProjectPath string
)

// SetupConfig creates the config directory and file if they are missing and loads them into viper.
func SetupConfig() {
	home, _ := os.UserHomeDir()
	LabDir = filepath.Join(home, ".config", "lab")
	err := os.MkdirAll(LabDir, utils.DirPerm)
	utils.Check(err)

	if ConfigPath == "" {
		ConfigPath = filepath.Join(LabDir, "config.toml")
	}

	if _, err = os.Stat(ConfigPath); os.IsNotExist(err) {
		err = os.WriteFile(ConfigPath, []byte(content), utils.FilePerm)
		utils.Check(err)
	}

	buf, err := envsubst.ReadFile(ConfigPath)
	utils.Check(err)
	viper.SetConfigType("toml")
	viper.AddConfigPath(LabDir)

	err = viper.ReadConfig(bytes.NewReader(buf))
	utils.Check(err)
}

type gitlabConfig struct {
	BaseURL   string   `mapstructure:"base_url"`
	Token     string   `mapstructure:"token"`
	Codespace string   `mapstructure:"codespace"`
	Name      string   `mapstructure:"name"`
	Email     string   `mapstructure:"email"`
	Projects  string   `mapstructure:"projects"`
	Groups    []string `mapstructure:"groups"`
}

type mainConfig struct {
	ThemeColor     string `mapstructure:"theme_color"`
	CloneOpts      string `mapstructure:"clone_opts"`
	TailLineNumber int64  `mapstructure:"tail_line_number"`
	FZF            bool   `mapstructure:"fzf"`
}

// Setup loads the gitlab and main sections of the config and validates the required values.
func Setup() {
	setupMainConfig()
	SetupGitLabConfig()

	if len(Config.BaseURL) == 0 {
		utils.Err("set Gitlab base url first, use `lab config`")
	}
}

func setupMainConfig() {
	MainConfig = &mainConfig{}

	viper.SetDefault("main.theme_color", "79")

	err := viper.Sub("main").Unmarshal(MainConfig)
	utils.Check(err)

	if len(MainConfig.ThemeColor) == 0 {
		MainConfig.ThemeColor = "79"
	}

	if MainConfig.TailLineNumber == 0 {
		MainConfig.TailLineNumber = 20
	}
}

// SetupGitLabConfig loads the gitlab section of the config and normalises its values.
// It needs no token, so commands can run before a token exists.
func SetupGitLabConfig() {
	Config = &gitlabConfig{}

	err := viper.Sub("gitlab").Unmarshal(Config)
	utils.Check(err)

	Config.BaseURL = normalizeBaseURL(Config.BaseURL)

	if len(Config.Groups) == 0 {
		Config.Groups = syncGroups()
	}

	home, err := os.UserHomeDir()
	utils.Check(err)

	codespace := Config.Codespace
	if strings.HasPrefix(codespace, "~") {
		codespace = filepath.Join(home, codespace[1:])
	}

	if strings.HasSuffix(codespace, string(os.PathSeparator)) {
		codespace = codespace[:len(codespace)-len(string(os.PathSeparator))]
	}

	Config.Codespace = codespace
	if Config.Projects == "" {
		Config.Projects = filepath.Join(LabDir, ".projects")
	}

	ProjectPath = Config.Projects
}

// normalizeBaseURL adds a https scheme when it is missing and drops a trailing slash.
func normalizeBaseURL(baseURL string) string {
	if baseURL == "" {
		return ""
	}

	if !strings.HasPrefix(baseURL, "http") {
		baseURL = "https://" + baseURL
	}

	return strings.TrimSuffix(baseURL, "/")
}
