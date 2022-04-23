package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

func (c *Config) runLocal(args []string) {
	f := flag.NewFlagSet("local", flag.ExitOnError)
	filesFlag := f.String("files", "", "colon-separated list of files to use as anchors for the search for the local file")
	positionFlag := f.Bool("pos", false, "if true, prints the zero-indexed line/offset, instead of the file")
	f.Usage = func() {
		fmt.Fprintln(os.Stderr, `Usage: sgbl local <url>

Translates a Sourcegraph URL into a local file path`)
		f.PrintDefaults()
	}
	if err := f.Parse(args); err != nil {
		f.Usage()
		os.Exit(1)
	}

	if *positionFlag {
		line, col, err := localPosition(f.Arg(0))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(1)
		}
		fmt.Printf("%d:%d\n", line, col)
		os.Exit(0)
	}

	var files []string
	if *filesFlag != "" {
		files = strings.Split(*filesFlag, ":")
	}

	if err := c.local(f.Arg(0), files); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

var posRegex = regexp.MustCompile(`^L([0-9]+)(?:\:([0-9]+))?$`)

func localPosition(rawURL string) (int, int, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return 0, 0, errors.Wrap(err, "failed to extract local position")
	}
	lp := u.Fragment
	matches := posRegex.FindStringSubmatch(lp)
	if len(matches) <= 1 {
		return 0, 0, nil
	}
	if len(matches) == 2 {
		line, err := strconv.Atoi(matches[1])
		if err != nil {
			return 0, 0, err
		}
		return line, 0, nil
	}
	if len(matches) == 3 {
		line, err := strconv.Atoi(matches[1])
		if err != nil {
			return 0, 0, err
		}
		if matches[2] == "" {
			return line, 0, nil
		}
		col, err := strconv.Atoi(matches[2])
		if err != nil {
			return 0, 0, err
		}
		return line, col, nil
	}
	return 0, 0, fmt.Errorf("Found more than 2 submatches: %+v", matches)
}

func (c *Config) local(rawURL string, anchorPaths []string) error {
	targetRepo, targetPath, err := extractRepoPath(rawURL)
	if err != nil {
		return err
	}

	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	for _, p := range append([]string{wd}, anchorPaths...) {
		finfo, err := os.Stat(p)
		if err != nil {
			continue
		}
		candidateRepo, err := evalRepoURI(p, finfo.IsDir())
		if err != nil {
			return err
		}
		candidateRepoRoot, err := evalAbsRepoRoot(p, finfo.IsDir())
		if err != nil {
			return err
		}
		if candidateRepo == targetRepo {
			fmt.Println(filepath.Join(candidateRepoRoot, filepath.Join(strings.Split(targetPath, "/")...)))
			return nil
		}
	}

	return errors.New("Could not find matching local file")
}

func extractRepoPath(rawURL string) (r, p string, err error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", "", errors.Wrap(err, "failed to extract repo path")
	}

	if idx := strings.Index(u.Path, "/-/blob/"); idx >= 0 {
		r, p = u.Path[1:idx], u.Path[idx+len("/-/blob/"):]
		if jdx := strings.Index(r, "@"); jdx >= 0 {
			r = r[:jdx]
		}
		return r, p, nil
	} else {
		return "", "", nil
	}
}
