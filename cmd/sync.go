package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

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

	ns, counts := internal.Projects()
	sort.Strings(ns)

	synced := 0

	for _, n := range ns {
		if n != "" {
			fmt.Fprintln(file, n)

			synced++
		}
	}

	fmt.Fprint(os.Stderr, groupCountsReport(counts))
	fmt.Fprintf(os.Stderr, "Done. %d projects\n", synced)
}

// groupCountsReport renders one "group count" line per group, aligned on the
// longest group path. counts must be sorted by group.
func groupCountsReport(counts []internal.GroupProjects) string {
	width := 0
	for _, c := range counts {
		width = max(width, len(c.Group))
	}

	var out strings.Builder

	for _, c := range counts {
		fmt.Fprintf(&out, "%-*s %d\n", width, c.Group, c.Projects)
	}

	return out.String()
}
