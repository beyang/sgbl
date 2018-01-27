package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
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
	searchQuery := flag.String("search", "", "a search query")
	posFlag := flag.String("pos", "", "the position you wish to open the file at")
	lineFlag := flag.String("line", "", "the line you wish to open the file at")
	colFlag := flag.String("col", "", "the column you wish to open the file at")

	flag.Parse()

	var err error

	cfg, err := readConfig()
	if err != nil {
		if err != nil {
			return fmt.Errorf("error reading config: %s", err)
		}
	}

	var pathArg string
	args := flag.Args()
	if len(args) == 1 {
		pathArg = args[0]
	}

	abspath, err := filepath.Abs(pathArg)
	if err != nil {
		return err
	}

	finfo, err := os.Lstat(abspath)
	if err != nil {
		return err
	}

	repoURI, err := evalRepoURI(abspath, finfo.IsDir())
	if err != nil {
		return err
	}

	relPath, err := evalRelPathFromRepoRoot(abspath, finfo.IsDir())
	if err != nil {
		return err
	}

	if relPath == "" && pathArg == "." {
		relPath = "/"
	}

	sgURL := evalSGURL(sgURLOptions{
		sgHost:  cfg.sourcegraphURLForRepo(repoURI),
		repoURI: repoURI,
		relPath: relPath,
		query:   *searchQuery,
		pos:     buildPos(*posFlag, *lineFlag, *colFlag),
	})

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
	defer func() {
		if err := cfgFile.Close(); err != nil {
			fmt.Errorf("unexpected error: %v", err)
		}
	}()

	var cfg Config
	if err := json.NewDecoder(cfgFile).Decode(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

type sgURLOptions struct {
	sgHost  string
	repoURI string
	relPath string
	query   string
	pos     string
	isDir   bool
}

func evalSGURL(opts sgURLOptions) string {
	var url string

	if opts.relPath == "" && opts.query != "" {
		url = fmt.Sprintf("%s/search", opts.sgHost)
	} else if opts.isDir {
		if opts.relPath == "" || opts.relPath == "/" {
			url = fmt.Sprintf("%s/%s", opts.sgHost, opts.repoURI)
		} else {
			url = fmt.Sprintf("%s/%s/-/tree/%s", opts.sgHost, opts.repoURI, opts.relPath)
		}
	} else {
		url = fmt.Sprintf("%s/%s/-/blob/%s", opts.sgHost, opts.repoURI, opts.relPath)
	}

	if opts.pos != "" {
		url += "#" + opts.pos
	}

	if opts.query != "" {
		url += "?" + buildSearchURLQuery(opts.query)
	}

	return url
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

func buildSearchURLQuery(query string) string {
	// Compile here not globally so we don't waste time if no search specified
	slashRe := regexp.MustCompile("%2F")
	colonRe := regexp.MustCompile("%3A")

	qs := make(url.Values)
	qs.Add("q", query)

	encoded := qs.Encode()
	encoded = slashRe.ReplaceAllString(encoded, "/")
	encoded = colonRe.ReplaceAllString(encoded, ";")

	return encoded
}

func buildPos(pos, line, col string) string {
	var p string

	if pos != "" {
		p = fmt.Sprintf("L%s", pos)
	} else if line != "" {
		if col != "" {
			p = fmt.Sprintf("L%s:%s", line, col)
		} else {
			p = fmt.Sprintf("L%s", line)
		}
	}

	return p
}
