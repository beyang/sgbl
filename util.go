package main

import (
	"bytes"
	"fmt"
	"net/url"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

func evalFilePlusURL(fileURL string, query string, pos string) string {
	url := fileURL
	if query != "" {
		url += "?" + evalSearchURLQuery(query)
	}
	if pos != "" {
		url += "#" + pos
	}
	return url
}

func evalFileURL(sgHost, repoURI, relPath string, isDir bool) string {
	if isDir {
		if relPath == "" || relPath == "/" {
			return fmt.Sprintf("%s/%s", sgHost, repoURI)
		}
		return fmt.Sprintf("%s/%s/-/tree/%s", sgHost, repoURI, relPath)
	}
	return fmt.Sprintf("%s/%s/-/blob/%s", sgHost, repoURI, relPath)
}

func evalAbsRepoRoot(path string, isDir bool) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	if isDir {
		cmd.Dir = path
	} else {
		cmd.Dir = filepath.Dir(path)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("unknown `git rev-parse --show-toplevel` error: %s, output was:\n%s", err, string(out))
	}
	return string(bytes.TrimSpace(out)), nil
}

// evalRelPathFromRepoRoot computes the relative path from the repository root by
// relativizing the abspath to the repository
// root, and converting OS-specific separators to "/".
func evalRelPathFromRepoRoot(abspath string, isDir bool) (relPath string, err error) {
	rootDir, err := evalAbsRepoRoot(abspath, isDir)
	if err != nil {
		return "", err
	}
	if rootDir == abspath {
		return "", nil
	}
	relPath, err = filepath.Rel(rootDir, abspath)
	if err != nil {
		return "", err
	}
	if relPath == ".." || strings.HasPrefix(relPath, fmt.Sprintf("..%c", filepath.Separator)) {
		return "", errors.New("file path points outside current repository")
	}
	return strings.Replace(relPath, string(filepath.Separator), "/", -1), nil
}

// evalRepoURI returns the Sourcegraph repository URI corresponding to the git repository that
// contains the file/directory specified by `abspath`.
//
// TODO: this should take into account `repositoryPathPattern`s.
func evalRepoURI(abspath string, isDir bool) (string, error) {
	var remotes []string
	{
		remotesRaw, err := exec.Command("git", "remote").CombinedOutput()
		if err != nil {
			return "", err
		}
		for _, r := range strings.Split(string(remotesRaw), "\n") {
			if r2 := strings.TrimSpace(r); r2 != "" {
				if r2 == "origin" {
					remotes = append([]string{"origin"}, remotes...)
				} else {
					remotes = append(remotes, r2)
				}
			}
		}
	}
	if len(remotes) == 0 {
		return "", errors.New("no git remote origin found")
	}

	var dir string
	if isDir {
		dir = abspath
	} else {
		dir = filepath.Dir(abspath)
	}
	var (
		rawRemoteURL string
		err          error
	)
	for _, remote := range remotes {
		rawRemoteURL, err = evalRepoURIWithRemote(dir, remote)
		if err == nil {
			break
		}
	}
	if err != nil {
		return "", err
	}

	return evalRepoURIFromRawRemoteURL(rawRemoteURL)
}

func evalRepoURIFromRawRemoteURL(rawRemoteURL string) (string, error) {
	if strings.HasPrefix(rawRemoteURL, "git@github.com:") {
		return strings.TrimSuffix(strings.Replace(rawRemoteURL, "git@github.com:", "github.com/", 1), ".git"), nil
	}
	remoteURL, err := url.Parse(rawRemoteURL)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse remote URL")
	}
	if remoteURL.Host == "github.com" && remoteURL.Scheme == "https" {
		return remoteURL.Host + strings.TrimSuffix(remoteURL.Path, ".git"), nil
	}
	return "", fmt.Errorf("unrecognized git repository host %q, supported ones are: github.com", remoteURL.Host)
}

func evalRepoURIWithRemote(dir, remote string) (string, error) {
	cmd := exec.Command("git", "config", "--get", fmt.Sprintf("remote.%s.url", remote))
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		if len(bytes.TrimSpace(out)) == 0 {
			return "", fmt.Errorf("no git remote %s found", remote)
		}
		return "", fmt.Errorf("unknown `git config --get remote.%s.url` error: %s, output was:\n%s", remote, err, string(out))
	}
	return strings.TrimSpace(string(out)), nil
}

func evalSearchURLQuery(query string) string {
	// Compile here not globally so we don't waste time if no search specified
	slashRe := regexp.MustCompile("%2F")
	colonRe := regexp.MustCompile("%3A")

	qs := make(url.Values)
	qs.Add("q", query)

	encoded := qs.Encode()
	encoded = slashRe.ReplaceAllString(encoded, "/")
	encoded = colonRe.ReplaceAllString(encoded, ":")

	return encoded
}

func (c *Config) sourcegraphURLForRepo(repoURI string) string {
	for _, sg := range c.Sourcegraphs {
		for _, r := range sg.Repos {
			if r == repoURI {
				return sg.URL
			}
		}
	}
	return "https://sourcegraph.com"
}
