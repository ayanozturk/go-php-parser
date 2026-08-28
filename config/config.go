package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/ayanozturk/go-php-parser/overrides"

	"gopkg.in/yaml.v2"
)

var DefaultConfigFilenames = []string{
	"tusk.yaml",
	"go-phpcs.yaml",
	"go-phpcs.yml",
	"config.yaml",
}

const DefaultConfigContent = `path: .
# Extra directories indexed for cross-file symbol resolution (e.g. vendor)
# but never scanned for diagnostics themselves.
includes: []
extensions:
  - php
ignore:
  - vendor
  - node_modules
  - cache
  - .git
rules:
  - all
analysis_level: 0
`

// WriteDefaultConfig creates filename with default config content.
// Returns an error if the file already exists.
func WriteDefaultConfig(filename string) error {
	if _, err := os.Stat(filename); err == nil {
		return fmt.Errorf("%s already exists", filename)
	} else if !os.IsNotExist(err) {
		return err
	}
	return os.WriteFile(filename, []byte(DefaultConfigContent), 0644)
}

type Config struct {
	Path          string                  `yaml:"path"`
	Includes      []string                `yaml:"includes"`
	Extensions    []string                `yaml:"extensions"`
	Ignore        []string                `yaml:"ignore"`
	Rules         []string                `yaml:"rules"`
	AnalysisLevel *int                    `yaml:"analysis_level"`
	Overrides     overrides.RuleOverrides `yaml:"overrides"`
}

func DiscoverConfig(dir string) (string, error) {
	if dir == "" {
		dir = "."
	}

	for _, name := range DefaultConfigFilenames {
		candidate := filepath.Join(dir, name)
		info, err := os.Stat(candidate)
		if err == nil {
			if info.IsDir() {
				continue
			}
			return candidate, nil
		}
		if !os.IsNotExist(err) {
			return "", err
		}
	}

	return "", fmt.Errorf("no config file found; checked %s", DefaultConfigFilenames)
}

func LoadConfig(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config Config
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, err
	}
	if config.AnalysisLevel != nil && *config.AnalysisLevel < 0 {
		return nil, fmt.Errorf("analysis_level must be zero or greater")
	}

	return &config, nil
}

func PrintEffectiveConfig(w io.Writer, cfg *Config, source string) {
	fmt.Fprintf(w, "config_file: %s\n", quoteYAMLString(source))
	fmt.Fprintf(w, "path: %s\n", quoteYAMLString(cfg.Path))
	writeStringList(w, "includes", cfg.Includes)
	writeStringList(w, "extensions", cfg.Extensions)
	writeStringList(w, "ignore", cfg.Ignore)
	writeStringList(w, "rules", cfg.Rules)
	if cfg.AnalysisLevel == nil {
		fmt.Fprintln(w, "analysis_level: null")
	} else {
		fmt.Fprintf(w, "analysis_level: %d\n", *cfg.AnalysisLevel)
	}
	writeOverrides(w, cfg.Overrides)
}

func writeStringList(w io.Writer, name string, values []string) {
	if len(values) == 0 {
		fmt.Fprintf(w, "%s: []\n", name)
		return
	}
	fmt.Fprintf(w, "%s:\n", name)
	for _, value := range values {
		fmt.Fprintf(w, "  - %s\n", quoteYAMLString(value))
	}
}

func writeOverrides(w io.Writer, ruleOverrides overrides.RuleOverrides) {
	if len(ruleOverrides) == 0 {
		fmt.Fprintln(w, "overrides: {}")
		return
	}

	codes := make([]string, 0, len(ruleOverrides))
	for code := range ruleOverrides {
		codes = append(codes, code)
	}
	sort.Strings(codes)

	fmt.Fprintln(w, "overrides:")
	for _, code := range codes {
		fmt.Fprintf(w, "  %s:\n", quoteYAMLString(code))
		writeNestedStringList(w, "classes", ruleOverrides[code].Classes)
	}
}

func writeNestedStringList(w io.Writer, name string, values []string) {
	if len(values) == 0 {
		fmt.Fprintf(w, "    %s: []\n", name)
		return
	}
	fmt.Fprintf(w, "    %s:\n", name)
	for _, value := range values {
		fmt.Fprintf(w, "      - %s\n", quoteYAMLString(value))
	}
}

func quoteYAMLString(value string) string {
	return strconv.Quote(value)
}

func ignoreDirSet(ignore []string) map[string]struct{} {
	set := make(map[string]struct{}, len(ignore))
	for _, dir := range ignore {
		set[dir] = struct{}{}
	}
	return set
}

func extSet(extensions []string) map[string]struct{} {
	set := make(map[string]struct{}, len(extensions))
	for _, ext := range extensions {
		set["."+ext] = struct{}{}
	}
	return set
}

func walkForFiles(root string, ignoreDirs, allowedExts map[string]struct{}) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip ignored directories
		if d.IsDir() {
			if _, ignored := ignoreDirs[d.Name()]; ignored {
				return filepath.SkipDir
			}
			return nil
		}

		// Check file extensions
		if _, allowed := allowedExts[filepath.Ext(path)]; allowed {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

func GetFilesToScan(config *Config) ([]string, error) {
	return walkForFiles(config.Path, ignoreDirSet(config.Ignore), extSet(config.Extensions))
}

// GetIncludeFiles walks the directories listed in config.Includes and
// returns their PHP files. These files are parsed and indexed for
// cross-file symbol resolution (e.g. vendor code) but are never scanned
// for diagnostics themselves.
func GetIncludeFiles(config *Config) ([]string, error) {
	ignoreDirs := ignoreDirSet(config.Ignore)
	allowedExts := extSet(config.Extensions)

	var files []string
	for _, dir := range config.Includes {
		found, err := walkForFiles(dir, ignoreDirs, allowedExts)
		if err != nil {
			return nil, err
		}
		files = append(files, found...)
	}
	return files, nil
}

// StreamFilesToScan walks the configured path in a background goroutine and
// streams discovered file paths into the returned channel, which is closed
// when the walk completes. This allows callers to overlap I/O and parsing
// with the directory walk rather than waiting for the full file list first.
func StreamFilesToScan(config *Config) <-chan string {
	ignoreDirs := ignoreDirSet(config.Ignore)
	allowedExts := extSet(config.Extensions)

	ch := make(chan string, 256)
	go func() {
		defer close(ch)
		filepath.WalkDir(config.Path, func(path string, d os.DirEntry, err error) error { //nolint:errcheck
			if err != nil {
				return nil // skip unreadable entries
			}
			if d.IsDir() {
				if _, ignored := ignoreDirs[d.Name()]; ignored {
					return filepath.SkipDir
				}
				return nil
			}
			if _, allowed := allowedExts[filepath.Ext(path)]; allowed {
				ch <- path
			}
			return nil
		})
	}()
	return ch
}
