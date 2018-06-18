package main

import (
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
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

// evalRelPathFromRepoRoot computes the relative path from the repository root by
// relativizing the abspath to the repository
// root, and converting OS-specific separators to "/".
func evalRelPathFromRepoRoot(abspath string, isDir bool) (relPath string, err error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	if isDir {
		cmd.Dir = abspath
	} else {
		cmd.Dir = filepath.Dir(abspath)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("unknown `git rev-parse --show-toplevel` error: %s, output was:\n%s", err, string(out))
	}
	rootDir := string(bytes.TrimSpace(out))
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

func evalRepoURI(abspath string, isDir bool) (string, error) {
	cmd := exec.Command("git", "config", "--get", "remote.origin.url")
	if isDir {
		cmd.Dir = abspath
	} else {
		cmd.Dir = filepath.Dir(abspath)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		if len(bytes.TrimSpace(out)) == 0 {
			return "", errors.New("no git remote origin found")
		}
		return "", fmt.Errorf("unknown `git config --get remote.origin.url` error: %s, output was:\n%s", err, string(out))
	}
	rawRemoteURL := strings.TrimSpace(string(out))

	var remoteURL *url.URL
	if strings.HasPrefix(rawRemoteURL, "git@github.com:") {
		remoteURL, err = url.Parse("//" + rawRemoteURL)
	} else {
		remoteURL, err = url.Parse(rawRemoteURL)
	}
	if err != nil {
		return "", err
	}

	var repoURI string
	switch {
	case strings.HasPrefix(remoteURL.Host, "github.com:") && remoteURL.User != nil && remoteURL.User.Username() == "git":
		repoURI = strings.Replace(remoteURL.Host, ":", "/", -1) + strings.TrimSuffix(remoteURL.Path, ".git")
	case remoteURL.Host == "github.com" && remoteURL.Scheme == "https":
		repoURI = remoteURL.Host + strings.TrimSuffix(remoteURL.Path, ".git")
	default:
		return "", errors.New("unrecognized git repository host, supported ones are: github.com")
	}

	return repoURI, nil
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
