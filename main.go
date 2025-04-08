package main

import (
	"flag"
	"fmt"
	"go-phpcs/command"
	"go-phpcs/lexer"
	"go-phpcs/parser"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Path       string   `yaml:"path"`
	Extensions []string `yaml:"extensions"`
	Ignore     []string `yaml:"ignore"`
}

func main() {
	// Load the configuration
	config, err := loadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	filesToScan, err := getFilesToScan(config)
	if err != nil {
		log.Fatalf("Error scanning files: %v", err)
	}

	// Print the files to scan
	// fmt.Println("Files to scan:")
	// for _, file := range filesToScan {
	// 	fmt.Println(file)
	// }

	debug := flag.Bool("debug", false, "Enable debug mode to show parsing errors")
	flag.Parse()

	if len(flag.Args()) < 2 && len(filesToScan) == 0 {
		fmt.Println("Usage: go-phpcs <command> <file>")
		command.PrintUsage()
		os.Exit(1)
	}

	commandName := "ast"
	if len(flag.Args()) > 0 {
		commandName = flag.Args()[0]
	}

	filePath := ""
	if len(flag.Args()) > 1 {
		filePath = flag.Args()[1]
	} else {
		filePath = filesToScan[0]
	}
	fmt.Println(filePath)
	if filePath == "" {
		fmt.Println("No file specified for parsing.")
		command.PrintUsage()
		os.Exit(1)
	}

	// Read the PHP file
	input, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Error reading file: %s\n", err)
		os.Exit(1)
	}

	// Create new lexer
	l := lexer.New(string(input))

	// Create new parser
	p := parser.New(l, *debug)

	// Parse the input
	nodes := p.Parse()

	// Check for parsing errors
	if len(p.Errors()) > 0 {
		fmt.Println("Parsing errors:")
		for _, err := range p.Errors() {
			fmt.Printf("\t%s\n", err)
		}
		os.Exit(1)
	}

	// Handle commands
	if cmd, exists := command.Commands[commandName]; exists {
		if commandName == "tokens" {
			// For tokens command, create a new lexer with the original input
			l := lexer.New(string(input))
			for {
				tok := l.NextToken()
				if tok.Type == "T_EOF" {
					break
				}
				fmt.Printf("%s: %s @ %d:%d\n", tok.Type, tok.Literal, tok.Pos.Line, tok.Pos.Column)
			}
		} else {
			cmd.Execute(nodes)
		}
	} else {
		fmt.Printf("Unknown command: %s\n", commandName)
		command.PrintUsage()
		os.Exit(1)
	}
}

func loadConfig(filename string) (*Config, error) {
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

func getFilesToScan(config *Config) ([]string, error) {
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
