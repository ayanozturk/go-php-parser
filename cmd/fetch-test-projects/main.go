// Command fetch-test-projects clones or verifies the pinned corpora listed
// in test_projects/manifest.json into test_projects/<name>. These corpora
// back cmd/compat-metrics and cmd/benchmark; they are large (tens to
// hundreds of MB each) and are intentionally not committed to this
// repository (see .gitignore) — every checkout must instead be
// reproducible from the manifest's pinned commit, per the
// comparable-performance contract in
// docs/full-static-analyser-target.md#comparable-performance-contract.
//
// Usage:
//
//	go run ./cmd/fetch-test-projects
//	go run ./cmd/fetch-test-projects --only psl,wordpress-develop
//	go run ./cmd/fetch-test-projects --force
//
// Each project directory is fetched by creating an empty git repository,
// adding the manifest's repo URL as "origin", fetching exactly the pinned
// commit at depth 1, and checking it out. This pins an exact commit (not
// just a branch or tag, which can move) while still avoiding a full clone
// of the upstream project's history. It requires the remote to allow
// fetching arbitrary commit SHAs (GitHub enables this by default for
// public repositories).
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type project struct {
	Name   string `json:"name"`
	Repo   string `json:"repo"`
	Ref    string `json:"ref"`
	Commit string `json:"commit"`
	Notes  string `json:"notes"`
}

type manifest struct {
	Projects []project `json:"projects"`
}

func main() {
	manifestPath := flag.String("manifest", "test_projects/manifest.json", "path to the corpus manifest")
	only := flag.String("only", "", "comma-separated subset of project names to fetch (default: all)")
	force := flag.Bool("force", false, "re-fetch even if the project directory already exists and matches the pinned commit")
	flag.Parse()

	data, err := os.ReadFile(*manifestPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fetch-test-projects: reading manifest: %v\n", err)
		os.Exit(1)
	}
	var m manifest
	if err := json.Unmarshal(data, &m); err != nil {
		fmt.Fprintf(os.Stderr, "fetch-test-projects: parsing manifest: %v\n", err)
		os.Exit(1)
	}

	var wanted map[string]bool
	if *only != "" {
		wanted = make(map[string]bool)
		for _, name := range strings.Split(*only, ",") {
			wanted[strings.TrimSpace(name)] = true
		}
	}

	root := filepath.Dir(*manifestPath)
	failures := 0
	for _, p := range m.Projects {
		if wanted != nil && !wanted[p.Name] {
			continue
		}
		dest := filepath.Join(root, p.Name)
		if err := fetchProject(p, dest, *force); err != nil {
			fmt.Fprintf(os.Stderr, "fetch-test-projects: %s: %v\n", p.Name, err)
			failures++
			continue
		}
		fmt.Printf("fetch-test-projects: %s ready at %s (%s)\n", p.Name, dest, p.Commit)
	}

	if failures > 0 {
		os.Exit(1)
	}
}

func fetchProject(p project, dest string, force bool) error {
	if p.Commit == "" {
		return fmt.Errorf("manifest entry has no pinned commit")
	}

	if info, err := os.Stat(dest); err == nil && info.IsDir() {
		if !force {
			if head, err := runGitOutput(dest, "rev-parse", "HEAD"); err == nil && strings.TrimSpace(head) == p.Commit {
				return nil // already at the pinned commit
			}
		}
		if err := os.RemoveAll(dest); err != nil {
			return fmt.Errorf("removing existing checkout: %w", err)
		}
	}

	if err := os.MkdirAll(dest, 0o755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}
	steps := [][]string{
		{"init", "-q"},
		{"remote", "add", "origin", p.Repo},
		{"fetch", "--depth", "1", "origin", p.Commit},
		{"checkout", "-q", "FETCH_HEAD"},
	}
	for _, args := range steps {
		if err := runGit(dest, args...); err != nil {
			return fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
		}
	}
	return nil
}

func runGit(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runGitOutput(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	return string(out), err
}
