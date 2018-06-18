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
	f.Parse(args)

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
	fmt.Fprintln(os.Stderr, "TODO: usage")
}

func (c *Config) runOpen(args []string) {
	f := flag.NewFlagSet("open", flag.ExitOnError)
	f.Usage = printUsage
	posFlag := f.String("pos", "", "the position at which to open the file, formatted as \"L${line}:${col}\"")
	if err := f.Parse(args); err != nil {
		printUsage()
		os.Exit(1)
	}

	pathArg := flag.Arg(0)
	if err := c.open(pathArg, *posFlag); err != nil {
		fmt.Fprintln(os.Stderr, "Error: %s", err)
		os.Exit(1)
	}
}

func (c *Config) runSearch(args []string) {
	f := flag.NewFlagSet("search", flag.ExitOnError)
	f.Usage = printUsage
	posFlag := f.String("pos", "", "the position at which to open the file, formatted as \"L${line}:${col}\"")
	pathFlag := f.String("path", "", "the path at which to make this search")
	if err := f.Parse(args); err != nil {
		printUsage()
		os.Exit(1)
	}
	query := f.Arg(0)
	if err := c.search(query, *pathFlag, *posFlag); err != nil {
		fmt.Fprintln(os.Stderr, "Error: %s", err)
		os.Exit(1)
	}
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
