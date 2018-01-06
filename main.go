package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
)

type Config struct {
	Sourcegraphs []SourcegraphInstance `json:"sourcegraphs"`
}

type SourcegraphInstance struct {
	URL   string   `json:"url"`
	Repos []string `json:"repos"`
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := readConfig()
	if err != nil {
		if err != nil {
			return fmt.Errorf("error reading config: %s", err)
		}
	}

	repoURI, err := evalRepoURI()
	if err != nil {
		return err
	}

	var subpath string
	if len(os.Args) == 2 {
		subpath = os.Args[1]
	}

	relPath, isDir, err := evalRelPathFromRepoRoot(subpath)
	if err != nil {
		return err
	}

	sgURL := evalSGURL(cfg.sourcegraphURLForRepo(repoURI), repoURI, relPath, isDir)
	switch runtime.GOOS {
	case "linux":
		if err := exec.Command("xdg-open", sgURL).Run(); err != nil {
			return fmt.Errorf("exec `xdg-open %s` failed: %s", sgURL, err)
		}
	case "darwin":
		if err := exec.Command("open", sgURL).Run(); err != nil {
			return fmt.Errorf("open %s failed: %s", sgURL, err)
		}
	default:
		return fmt.Errorf("OS %s unsupported", runtime.GOOS)
	}
	return nil
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

func readConfig() (*Config, error) {
	usr, err := user.Current()
	if err != nil {
		return nil, err
	}
	cfgFile, err := os.Open(filepath.Join(usr.HomeDir, ".sg-config"))
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}
	defer cfgFile.Close()

	var cfg Config
	if err := json.NewDecoder(cfgFile).Decode(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func evalSGURL(sgHost, repoURI, relPath string, isDir bool) string {
	if isDir {
		if relPath == "" || relPath == "/" {
			return fmt.Sprintf("%s/%s", sgHost, repoURI)
		}
		return fmt.Sprintf("%s/%s/-/tree/%s", sgHost, repoURI, relPath)
	}
	return fmt.Sprintf("%s/%s/-/blob/%s", sgHost, repoURI, relPath)
}

// evalRelPathFromRepoRoot computes the relative path from the repository root by finding
// the absolute path indicated by $PWD/$subpath, taking that relative to the repository
// root, and converting OS-specific separators to "/". Also returns whether the path
// indicated is a directory.
func evalRelPathFromRepoRoot(subpath string) (relPath string, isDir bool, err error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").CombinedOutput()
	if err != nil {
		return "", false, fmt.Errorf("unknown `git rev-parse --show-toplevel` error: %s, output was:\n%s", err, string(out))
	}
	rootDir := string(bytes.TrimSpace(out))

	pwd, err := os.Getwd()
	if err != nil {
		return "", false, err
	}

	fp := filepath.Clean(filepath.Join(pwd, subpath))
	if rootDir == fp {
		return "", true, nil
	}
	relPath, err = filepath.Rel(rootDir, fp)
	if err != nil {
		return "", false, err
	}
	if relPath == ".." || strings.HasPrefix(relPath, fmt.Sprintf("..%c", filepath.Separator)) {
		return "", false, errors.New("file path points outside current repository")
	}
	relPath = strings.Replace(relPath, string(filepath.Separator), "/", -1)

	if subpath == "" {
		return relPath, false, nil
	}

	fileinfo, err := os.Lstat(subpath)
	if err != nil {
		return "", false, err
	}
	return relPath, fileinfo.IsDir(), nil
}

func evalRepoURI() (string, error) {
	out, err := exec.Command("git", "config", "--get", "remote.origin.url").CombinedOutput()
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
