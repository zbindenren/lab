package internal

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/ackerr/lab/utils"
)

// GitCommand runs git with args and returns its stdout.
func GitCommand(args ...string) (string, error) {
	cmd := exec.Command("git", args...) // #nosec G204 -- git is invoked with controlled arguments
	cmd.Stderr = os.Stderr
	output, err := cmd.Output()

	return string(output), err
}

// Clone git clone the gitlab project
func Clone(gitURL, path string) error {
	args := []string{"clone", gitURL, path}

	extra := strings.SplitSeq(MainConfig.CloneOpts, " ")
	for a := range extra {
		if a != "" {
			args = append(args, a)
		}
	}

	cmd := exec.Command("git", args...) // #nosec G204 -- git is invoked with controlled arguments
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	err := cmd.Run()
	if err != nil {
		// if clone failed, remove the empty directory
		_ = os.Remove(path)
		return err
	}

	if len(Config.Email) > 0 {
		_ = SetGitConfig("email", Config.Email, path)
	}

	if len(Config.Name) > 0 {
		_ = SetGitConfig("name", Config.Name, path)
	}

	return err
}

// Fetch fetches all remotes of the repository at path.
func Fetch(path string) error {
	cmd := exec.Command("git", "-C", path, "fetch", "--all") // #nosec G204 -- git is invoked with a cleaned repository path
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err := cmd.Run()

	return err
}

// SetGitConfig sets user.<key> to value in the repository at path.
func SetGitConfig(key, value, path string) error {
	args := []string{}
	if len(path) > 0 {
		args = append(args, "-C", path)
	}

	args = append(args, "config", "user."+key, value)
	_, err := GitCommand(args...)

	return err
}

// CurrentGitRepo return the GitRepo path
func CurrentGitRepo() (string, error) {
	output, err := GitCommand("rev-parse", "-q", "--show-toplevel")
	return output, err
}

// CurrentBranch returns the current git branch, master if it cannot be resolved.
func CurrentBranch() string {
	branch, err := SymbolicRef("HEAD", true)
	if branch == "" || err != nil {
		branch = "master"
	}

	return branch
}

// RemoteURL returns the url of the given git remote.
func RemoteURL(remote string) string {
	gitURL, err := GitCommand("ls-remote", "--get-url", remote)
	if err != nil {
		utils.Err("git remote is not set for", remote)
	}

	gitURL = firstLine(gitURL)
	if gitURL == remote {
		utils.Err(remote, "is a wrong remote")
	}

	return gitURL
}

// CurrentRemote returns the remote tracked by branch, origin if it cannot be resolved.
func CurrentRemote(branch string) string {
	remote, err := GitCommand("config", fmt.Sprintf("branch.%s.remote", branch))

	remote = firstLine(remote)
	if remote == "" || err != nil {
		remote = "origin"
	}

	return remote
}

// SymbolicRef return the ref branch
func SymbolicRef(ref string, short bool) (string, error) {
	args := []string{"symbolic-ref"}
	if short {
		args = append(args, "--short")
	}

	args = append(args, ref)
	output, err := GitCommand(args...)

	return firstLine(output), err
}

// the git command output always has the "/n"
func firstLine(output string) string {
	if before, _, ok := strings.Cut(output, "\n"); ok {
		return before
	}

	return output
}

// TransferGitURLToURL example:
// git@github.com/Ackerr:lab.git     -> https://github.com/Ackerr/lab
// https://github.com/Ackerr/lab.git -> https://github.com/Ackerr/lab
func TransferGitURLToURL(gitURL string) string {
	var url string
	if strings.HasPrefix(gitURL, "https://") {
		url = gitURL[:len(gitURL)-4]
	}

	if strings.HasPrefix(gitURL, "git@") {
		url = gitURL[:len(gitURL)-4]
		url = strings.Replace(url, ":", "/", 1)
		url = strings.Replace(url, "git@", "https://", 1)
	}

	return url
}
