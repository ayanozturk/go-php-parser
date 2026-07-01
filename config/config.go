package config

import (
	"fmt"
	"os"
	"path/filepath"

	"go-phpcs/overrides"

	"gopkg.in/yaml.v2"
)

var DefaultConfigFilenames = []string{
	"go-phpcs.yaml",
	"go-phpcs.yml",
	"config.yaml",
}

type Config struct {
	Path          string                  `yaml:"path"`
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

	return &config, nil
}

func GetFilesToScan(config *Config) ([]string, error) {
	var filesToScan []string
	ignoreDirs := make(map[string]struct{}, len(config.Ignore))
	for _, ignore := range config.Ignore {
		ignoreDirs[ignore] = struct{}{}
	}
	allowedExts := make(map[string]struct{}, len(config.Extensions))
	for _, ext := range config.Extensions {
		allowedExts["."+ext] = struct{}{}
	}

	err := filepath.WalkDir(config.Path, func(path string, d os.DirEntry, err error) error {
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
			filesToScan = append(filesToScan, path)
		}

		return nil
	})

	return filesToScan, err
}

// StreamFilesToScan walks the configured path in a background goroutine and
// streams discovered file paths into the returned channel, which is closed
// when the walk completes. This allows callers to overlap I/O and parsing
// with the directory walk rather than waiting for the full file list first.
func StreamFilesToScan(config *Config) <-chan string {
	ignoreDirs := make(map[string]struct{}, len(config.Ignore))
	for _, ignore := range config.Ignore {
		ignoreDirs[ignore] = struct{}{}
	}
	allowedExts := make(map[string]struct{}, len(config.Extensions))
	for _, ext := range config.Extensions {
		allowedExts["."+ext] = struct{}{}
	}

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
