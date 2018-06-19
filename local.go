package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
)

func (c *Config) runLocal(args []string) {
	f := flag.NewFlagSet("local", flag.ExitOnError)
	f.Usage = func() {
		fmt.Fprintln(os.Stderr, `Usage: sg local <url>

Translates a Sourcegraph URL into a local file path
`)
		f.PrintDefaults()
	}

	if err := c.local(f.Arg(0)); err != nil {
		fmt.Fprintln(os.Stderr, "Error: %s", err)
		os.Exit(1)
	}
}

func (c *Config) local(rawURL string) error {
	u, err := url.Parse(rawlURL)
	if err != nil {
		return err
	}

	// TODO: use repositoryPathPattern?

	// Compute repository clone URL(s) (include all cloneURL candidates) and file path
	// - File path is easy (do this first)
	// - Repository: get repo URI
	//   - Hit API endpoint to resolve repo URI to clone URL
	//   - Convert cloneURL to variant forms

	// Search for files in following order:
	// - Explicit file candidates (editor extension passes in files of all open buffers)
	// - In repositories of explicit file candidates
	// - From sg-search-root (set in config)
	// - Print "could not find files in following locations"
}
