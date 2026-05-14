package config

import (
	"os"
	"path/filepath"

	"go-phpcs/overrides"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Path       string                  `yaml:"path"`
	Extensions []string                `yaml:"extensions"`
	Ignore     []string                `yaml:"ignore"`
	Rules      []string                `yaml:"rules"`
	Overrides  overrides.RuleOverrides `yaml:"overrides"`
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
