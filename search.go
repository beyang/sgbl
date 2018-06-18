package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

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
