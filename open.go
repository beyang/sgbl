package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func (c *Config) runOpen(args []string) {
	f := flag.NewFlagSet("open", flag.ExitOnError)
	f.Usage = func() {
		fmt.Fprintln(os.Stderr, `Usage: sg open [-pos POS] [-copy] [-print-url] <file>`)
		f.PrintDefaults()
	}
	posFlag := f.String("pos", "", "the position at which to open the file, formatted as \"L${line}:${col}\"")
	copyOnlyFlag := f.Bool("copy", false, "if set, then the URL will be copied to the clipboard instead of opened")
	urlOnlyFlag := f.Bool("print-url", false, "if set, then the URL will be printed instead of opened")
	if err := f.Parse(args); err != nil {
		f.Usage()
		os.Exit(1)
	}

	pathArg := f.Arg(0)
	if err := c.open(pathArg, *posFlag, *copyOnlyFlag, *urlOnlyFlag); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

func (c *Config) getSourcegraphURL(pathArg string, pos string) (string, error) {
	abspath, err := filepath.Abs(pathArg)
	if err != nil {
		return "", err
	}
	finfo, err := os.Lstat(abspath)
	if err != nil {
		return "", err
	}
	repoURI, err := evalRepoURI(abspath, finfo.IsDir())
	if err != nil {
		return "", err
	}
	relPath, err := evalRelPathFromRepoRoot(abspath, finfo.IsDir())
	if err != nil {
		return "", err
	}
	sgURL := evalFilePlusURL(
		evalFileURL(c.sourcegraphURLForRepo(repoURI), repoURI, relPath, finfo.IsDir()),
		"",
		pos,
	)
	return sgURL, nil
}

func (c *Config) open(pathArg string, pos string, copyOnly bool, printOnly bool) error {
	sgURL, err := c.getSourcegraphURL(pathArg, pos)
	if err != nil {
		return err
	}

	if printOnly {
		fmt.Println(sgURL)
		return nil
	}

	if copyOnly {
		switch runtime.GOOS {
		case "linux":
			cmd := exec.Command("xsel", "-ib")
			cmd.Stdin = bytes.NewBufferString(sgURL)
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("`xsel -ib` failed: %s", err)
			}
		case "darwin":
			cmd := exec.Command("pbcopy")
			cmd.Stdin = bytes.NewBufferString(sgURL)
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("`pbcopy` failed: %s", err)
			}
		default:
			return fmt.Errorf("OS %s unsupported", runtime.GOOS)
		}
		return nil
	}

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
