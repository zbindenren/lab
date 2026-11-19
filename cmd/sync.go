package cmd

import (
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"

	"github.com/ackerr/lab/internal"
	"github.com/ackerr/lab/utils"
)

func init() {
	rootCmd.AddCommand(syncCmd)
}

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync gitlab projects",
	Run: func(_ *cobra.Command, _ []string) {
		syncProjects()
	},
}

// 同步项目, 顺便按字母排个序
func syncProjects() {
	internal.Setup()

	file, err := os.Create(internal.ProjectPath)
	utils.Check(err)

	defer file.Close()

	ns := internal.Projects()
	sort.Strings(ns)

	for _, n := range ns {
		if n != "" {
			fmt.Fprintln(file, n)
		}
	}

	println("Done.")
}
