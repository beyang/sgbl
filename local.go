package main

import (
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

func (c *Config) runLocal(args []string) {
	f := flag.NewFlagSet("local", flag.ExitOnError)
	filesFlag := f.String("files", "", "colon-separated list of files to use as anchors for the search for the local file")
	f.Usage = func() {
		fmt.Fprintln(os.Stderr, `Usage: sg local <url>

Translates a Sourcegraph URL into a local file path
`)
		f.PrintDefaults()
	}
	if err := f.Parse(args); err != nil {
		f.Usage()
		os.Exit(1)
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
		return "", "", err
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
