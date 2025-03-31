package internal

import (
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/briandowns/spinner"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/ackerr/lab/utils"
)

var (
	perPage    = 100
	apiVersion = "v4"
)

const (
	interval = 3 * time.Second
	throttle = 50 * time.Microsecond
)

func NewClient() *gitlab.Client {
	path := gitlab.WithBaseURL(strings.Join([]string{Config.BaseURL, "api", apiVersion}, "/"))
	client, err := gitlab.NewClient(Config.Token, path)
	if err != nil {
		utils.Err(err)
	}
	return client
}

// Projects will return all projects path with namespace
func Projects(syncAll bool) []string {
	client := NewClient()

	return getAllGroupProjects(client, "linux", "kubernetes")
}

func projectNameSpaces(projects []*gitlab.Project) []string {
	ns := make([]string, 0, len(projects))
	for _, p := range projects {
		if p.Namespace.Kind == "group" {
			ns = append(ns, p.PathWithNamespace)
		}
	}
	return ns
}

// getAllProjects gets all projects. It performs numWorkers parallel requests. It starts a spinner until
// all projects are synchronized
func getAllProjects(client *gitlab.Client, syncAll bool, numWorkers int) []string {
	opt := gitlab.ListOptions{
		PerPage: perPage,
		Page:    1,
	}
	fmt.Println(numWorkers)

	projectsOpt := gitlab.ListProjectsOptions{
		Simple:           gitlab.Ptr(true),
		Membership:       gitlab.Ptr(!syncAll),
		SearchNamespaces: gitlab.Ptr(true),
		Search:           gitlab.Ptr("linux"),
		ListOptions:      opt,
	}
	// projectsOpt := gitlab.ListProjectsOptions{Simple: gitlab.Ptr(true), Membership: gitlab.Ptr(!syncAll), ListOptions: opt}

	// Start spinner
	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Prefix = "sync in process"
	s.Start()
	defer s.Stop()

	allProjects := []string{}
	for {
		ps, resp, err := client.Projects.ListProjects(&projectsOpt)
		if err != nil {
			fmt.Println(err)
			utils.PrintErr(err)
		}

		allProjects = append(allProjects, projectNameSpaces(ps)...)
		fmt.Println(len(allProjects), projectNameSpaces(ps)[1])
		fmt.Println(resp.NextPage)
		if resp.NextPage == 0 {
			break
		}
		projectsOpt.Page = resp.NextPage
	}

	return allProjects
}

// getAllGroupProjects gets all projects for a specific group and all its subgroups with pagination
func getAllGroupProjects(client *gitlab.Client, groups ...any) []string {
	allGroups := []any{}

	for _, g := range groups {
		allGroups = append(allGroups, g)
		allGroups = append(allGroups, getAllSubgroups(client, g)...)
	}

	// Get projects for each group
	var allProjects []string
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, gID := range allGroups {
		wg.Add(1)
		go func(gID any) {
			defer wg.Done()
			projects := getGroupProjects(client, gID)

			mu.Lock()
			allProjects = append(allProjects, projects...)
			mu.Unlock()

			time.Sleep(throttle) // Avoid rate limiting
		}(gID)
	}

	wg.Wait()
	return allProjects
}

// getGroupProjects gets projects for a single group with pagination
func getGroupProjects(client *gitlab.Client, groupID any) []string {
	opt := gitlab.ListOptions{
		PerPage: perPage,
		Page:    1,
	}

	var projects []string
	for {
		ps, resp, err := client.Groups.ListGroupProjects(groupID, &gitlab.ListGroupProjectsOptions{ListOptions: opt})
		if err != nil {
			utils.PrintErr(err)
			break
		}

		// Extract path with namespace for each project
		for _, p := range ps {
			projects = append(projects, p.PathWithNamespace)
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return projects
}

// getAllSubgroups gets all subgroups recursively for a specific group
func getAllSubgroups(client *gitlab.Client, groupID any) []any {
	opt := gitlab.ListOptions{
		PerPage: perPage,
		Page:    1,
	}

	var allSubgroups []any
	for {
		// Use ListDescendantGroups to get all descendant groups (including nested subgroups)
		subgroups, resp, err := client.Groups.ListDescendantGroups(groupID, &gitlab.ListDescendantGroupsOptions{
			ListOptions: opt,
		})
		if err != nil {
			utils.PrintErr(err)
			break
		}

		for _, sg := range subgroups {
			allSubgroups = append(allSubgroups, sg.ID)
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return allSubgroups
}

// TransferGitURLToProject example:
// git@gitlab.com/Ackerr:lab.git     -> Ackerr/lab
// https://gitlab.com/Ackerr/lab.git -> Ackerr/lab
func TransferGitURLToProject(gitURL string) string {
	var url string
	if strings.HasPrefix(gitURL, "https://") {
		url = gitURL[len(Config.BaseURL) : len(gitURL)-4]
	}
	if strings.HasPrefix(gitURL, "git@") {
		url = gitURL[:len(gitURL)-4]
		url = strings.Split(url, ":")[1]
	}
	return url
}

func TraceRunningJobs(client *gitlab.Client, pid any, jobs []*gitlab.Job, tailLine int64) bool {
	wg := sync.WaitGroup{}
	allDone := true
	for _, job := range jobs {
		if !IsRunning(job.Status) {
			continue
		}
		allDone = false
		wg.Add(1)
		go func(j *gitlab.Job) {
			_ = DoTrace(client, pid, j, tailLine)
			wg.Done()
		}(job)
	}
	wg.Wait()
	return allDone
}

// IsRunning check job status
func IsRunning(status string) bool {
	if status == "created" || status == "pending" || status == "running" {
		return true
	}
	return false
}

// gitlab trace log has some hidden content like this ^[[0m^[[0K^[[36;1mStart^[[0;m`
// It will make prefix failure, so replace it. PS: \x1b == ^[
var re = regexp.MustCompile(`\x1b\[0m.*\[0K`)

func DoTrace(client *gitlab.Client, pid any, job *gitlab.Job, tailLine int64) error {
	var offset int64
	firstTail := true
	prefix := utils.RandomColor(fmt.Sprintf("[%s] ", job.Name))
	for range time.NewTicker(interval).C {
		trace, _, err := client.Jobs.GetTraceFile(pid, job.ID)
		utils.Check(err)
		buffer, err := io.ReadAll(trace)
		utils.Check(err)
		lines := strings.Split(string(buffer), "\n")
		length := len(lines)
		if firstTail {
			begin := max(int64(length)-tailLine, 0)
			lines = lines[begin : len(lines)-1]
			firstTail = false
		}
		for _, line := range lines[offset:] {
			println(re.ReplaceAllString(prefix+line, ``))
		}
		offset = int64(length)
		if !IsRunning(job.Status) {
			return nil
		}
		job, _, err = client.Jobs.GetJob(pid, job.ID)
		utils.Check(err)
	}
	return nil
}
