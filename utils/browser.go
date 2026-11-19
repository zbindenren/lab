// Package utils provides helpers for files, colors, errors and processes.
package utils

import (
	"errors"
	"os"
	"os/exec"
	"runtime"

	"github.com/kballard/go-shellquote"
)

// return the browser launcher command, copy from github.com/github/hub
func browserLauncher() ([]string, error) {
	browser := os.Getenv("BROWSER")
	if browser == "" {
		browser = searchBrowserLauncher(runtime.GOOS)
	} else {
		browser = os.ExpandEnv(browser)
	}

	if browser == "" {
		return nil, errors.New("please set $BROWSER to a web launcher")
	}

	return shellquote.Split(browser)
}

func searchBrowserLauncher(goos string) (browser string) {
	switch goos {
	case "darwin":
		browser = "open"
	case "windows":
		browser = "cmd /c start"
	case "linux":
		browser = "xdg-open"
	default:
		browser = ""
	}

	return browser
}

// OpenBrowser open the url in web browser
func OpenBrowser(url string) error {
	launcher, err := browserLauncher()
	if err != nil {
		return err
	}

	args := make([]string, 0, len(launcher)+1)
	args = append(args, launcher...)
	args = append(args, url)

	cmd := exec.Command(args[0], args[1:]...) // #nosec G204 -- the launcher comes from $BROWSER or a known platform default

	return cmd.Run()
}
