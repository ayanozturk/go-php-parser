package main

import (
	"bufio"
	"fmt"
	"github.com/ayanozturk/go-php-parser/command"
	"github.com/ayanozturk/go-php-parser/config"
	"github.com/ayanozturk/go-php-parser/helper"
	_ "net/http/pprof" // registers /debug/pprof handlers on DefaultServeMux
	"os"
	"time"
)

func main() {
	os.Exit(run())
}

func run() int {
	args := helper.ParseCLIArgs(nil)
	outWriter := helper.SetupOutputFile(args)
	defer func() {
		// Flush buffered writer if applicable, then close.
		if bw, ok := outWriter.(*bufio.Writer); ok {
			bw.Flush()
		}
		if f, ok := outWriter.(*os.File); ok && f != os.Stdout {
			f.Close()
		}
	}()

	if args.CommandName == "list-style-rules" {
		command.Commands["list-style-rules"].Execute(nil, "", outWriter)
		return 0
	}
	if args.CommandName == "init" {
		command.Commands["init"].Execute(nil, "", outWriter)
		return 0
	}
	if _, exists := command.Commands[args.CommandName]; !exists {
		fmt.Fprintf(outWriter, "Unknown command: %s\n", args.CommandName)
		command.PrintUsageTo(outWriter)
		return 2
	}

	configPath := args.ConfigPath
	if configPath == "" {
		discovered, err := config.DiscoverConfig(".")
		if err != nil {
			fmt.Fprintf(outWriter, "Error discovering config: %v\n", err)
			return 2
		}
		configPath = discovered
	}

	c, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Fprintf(outWriter, "Error loading config: %v\n", err)
		return 2
	}

	if args.CommandName == "config" {
		config.PrintEffectiveConfig(outWriter, c, configPath)
		return 0
	}

	var filesToScan []string
	// analyze always needs the whole project indexed for cross-file symbol
	// resolution, even when a single target file is given.
	if !args.HasExplicitFile() || args.CommandName == "analyze" {
		filesToScan, err = config.GetFilesToScan(c)
		if err != nil {
			fmt.Fprintf(outWriter, "Error scanning files: %v\n", err)
			return 2
		}
	}

	if args.CommandName == "list-files" {
		helper.PrintFileList(outWriter, filesToScan)
		return 0
	}

	if !args.HasExplicitFile() && len(filesToScan) == 0 {
		fmt.Fprintln(outWriter, "Usage: go-phpcs <command> <file>")
		command.PrintUsageTo(outWriter)
		return 2
	}
	fmt.Fprintln(outWriter, "Command:", args.CommandName)

	stopProfiling := helper.SetupProfiling(args.Profile)
	defer stopProfiling()

	start := time.Now()
	var mem helper.MemStats
	helper.TrackMemoryUsage(&mem, true)
	outcome := helper.RunScanOrCommand(args, c, filesToScan, outWriter, &mem)
	helper.TrackMemoryUsage(&mem, false)
	elapsed := time.Since(start).Seconds()
	if args.CommandName != "analyze" {
		helper.PrintSummary(outWriter, outcome.TotalParseErrors, outcome.TotalLines, elapsed, mem)
	}
	return outcome.ExitCode
}
