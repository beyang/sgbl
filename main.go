package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
)

type Config struct {
	Sourcegraphs []SourcegraphInstance `json:"sourcegraphs"`
}

type SourcegraphInstance struct {
	URL   string   `json:"url"`
	Repos []string `json:"repos"`
}

func main() {
	runRoot(os.Args[1:])
}

func runRoot(args []string) {
	f := flag.NewFlagSet("root", flag.ExitOnError)
	f.Usage = printUsage
	if err := f.Parse(args); err != nil {
		f.Usage()
		os.Exit(1)
	}

	cfg, err := readConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not read config: %s\n", err)
		os.Exit(1)
	}

	switch f.Arg(0) {
	case "open":
		cfg.runOpen(args[1:])
	case "search":
		cfg.runSearch(args[1:])
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `sg {open|search} ...`)
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
