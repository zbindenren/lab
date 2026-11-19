package cmd

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/ackerr/lab/internal"
	"github.com/ackerr/lab/utils"
)

func init() {
	rootCmd.AddCommand(lintCmd)
}

var lintCmd = &cobra.Command{
	Use:   "lint",
	Short: "Check .gitlab-ci.yml syntax",
	Run:   lint,
}

// asdfasdf
func lint(_ *cobra.Command, args []string) {
	internal.Setup()
	var path string
	var err error
	if len(args) > 0 {
		path = args[0]
		path, err = filepath.Abs(path)
	} else {
		path, err = os.Getwd()
	}
	utils.Check(err)
	if !strings.HasSuffix(path, ".gitlab-ci.yml") {
		path = filepath.Join(path, ".gitlab-ci.yml")
	}
	if !utils.FileExists(path) {
		utils.Err(path, "not exist")
	}
	content, err := os.ReadFile(path)
	utils.Check(err)
	if len(content) == 0 {
		utils.Err("empty .gitlab-ci.yml")
	}
	client := internal.NewClient()
	// The project-less /ci/lint endpoint is gone, so validate in the context of
	// the project the current checkout's origin remote points to.
	project := strings.TrimPrefix(internal.TransferGitURLToProject(internal.RemoteURL("origin")), "/")
	result, _, err := client.Validate.ProjectNamespaceLint(project, &gitlab.ProjectNamespaceLintOptions{
		Content: gitlab.Ptr(string(content)),
	})
	utils.Check(err)
	if !result.Valid {
		for _, e := range result.Errors {
			utils.PrintlnWithColor(utils.ColorBg("ERROR", "#F08080"), e)
		}
	}
}
