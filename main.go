package main

import (
	"flag"
	"fmt"
	"go-phpcs/command"
	"go-phpcs/config"
	"go-phpcs/helper"
	"log"
	"os"
	"time"
)

func main() {
	c, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	filesToScan, err := config.GetFilesToScan(c)
	if err != nil {
		log.Fatalf("Error scanning files: %v", err)
	}

	args := helper.ParseCLIArgs(filesToScan)
	outWriter := helper.SetupOutputFile(args)
	defer func() {
		if f, ok := outWriter.(*os.File); ok && f != os.Stdout {
			f.Close()
		}
	}()
	if len(flag.Args()) < 2 && len(filesToScan) == 0 {
		fmt.Fprintln(outWriter, "Usage: go-phpcs <command> <file>")
		command.PrintUsage()
		os.Exit(1)
	}
	fmt.Fprintln(outWriter, "Command:", args.CommandName)

	stopProfiling := helper.SetupProfiling(args.Profile)
	defer stopProfiling()

	start := time.Now()
	var mem helper.MemStats
	helper.TrackMemoryUsage(&mem, true)
	totalParseErrors, totalLines := helper.RunScanOrCommand(args, c, filesToScan, outWriter, &mem)
	helper.TrackMemoryUsage(&mem, false)
	elapsed := time.Since(start).Seconds()
	helper.PrintSummary(outWriter, totalParseErrors, totalLines, elapsed, mem)
}
