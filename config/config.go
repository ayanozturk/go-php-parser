package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Path       string   `yaml:"path"`
	Extensions []string `yaml:"extensions"`
	Ignore     []string `yaml:"ignore"`
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

	err := filepath.Walk(config.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip ignored directories
		for _, ignore := range config.Ignore {
			if info.IsDir() && info.Name() == ignore {
				return filepath.SkipDir
			}
		}

		// Check file extensions
		if !info.IsDir() {
			for _, ext := range config.Extensions {
				if filepath.Ext(path) == "."+ext {
					filesToScan = append(filesToScan, path)
					break
				}
			}
		}

		return nil
	})

	return filesToScan, err
}
