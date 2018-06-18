package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func (c *Config) runSearch(args []string) {
	f := flag.NewFlagSet("search", flag.ExitOnError)
	f.Usage = func() {
		fmt.Fprintln(os.Stderr, `Usage: sg search [-pos POS] [-path PATH] <query>`)
		f.PrintDefaults()
	}
	posFlag := f.String("pos", "", "the position at which to open the file, formatted as \"L${line}:${col}\"")
	pathFlag := f.String("path", "", "the path at which to make this search")
	if err := f.Parse(args); err != nil {
		f.Usage()
		os.Exit(1)
	}
	query := f.Arg(0)
	if err := c.search(query, *pathFlag, *posFlag); err != nil {
		fmt.Fprintln(os.Stderr, "Error: %s", err)
		os.Exit(1)
	}
}

func (c *Config) search(query string, pathArg string, pos string) error {
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
	sgURL := evalFilePlusURL(
		evalFileURL(c.sourcegraphURLForRepo(repoURI), repoURI, relPath, finfo.IsDir()),
		query,
		pos,
	)
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
