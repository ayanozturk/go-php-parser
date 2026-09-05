package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ayanozturk/go-php-parser/analyse"
	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/lexer"
	"github.com/ayanozturk/go-php-parser/parser"
)

const reportSchemaVersion = 1

type manifest struct {
	SchemaVersion int               `json:"schemaVersion"`
	Reference     manifestReference `json:"reference"`
	Cases         []manifestCase    `json:"cases"`
}

type manifestReference struct {
	Tool  string `json:"tool"`
	Level int    `json:"level"`
}

type manifestCase struct {
	ID                 string   `json:"id"`
	Capability         string   `json:"capability"`
	File               string   `json:"file"`
	EngineCodes        []string `json:"engineCodes"`
	PHPStanIdentifiers []string `json:"phpstanIdentifiers"`
}

type differentialReport struct {
	SchemaVersion int          `json:"schemaVersion"`
	Engine        string       `json:"engine"`
	Reference     *toolReport  `json:"reference,omitempty"`
	Cases         []caseReport `json:"cases"`
	Totals        reportTotals `json:"totals"`
}

type toolReport struct {
	Tool    string `json:"tool"`
	Version string `json:"version"`
	Level   int    `json:"level"`
}

type caseReport struct {
	ID                   string   `json:"id"`
	Capability           string   `json:"capability"`
	File                 string   `json:"file"`
	ExpectedEngine       []string `json:"expectedEngine"`
	ActualEngine         []string `json:"actualEngine"`
	EngineMatches        bool     `json:"engineMatches"`
	ExpectedReference    []string `json:"expectedReference,omitempty"`
	ActualReference      []string `json:"actualReference,omitempty"`
	ReferenceMatches     *bool    `json:"referenceMatches,omitempty"`
	DifferentialConforms *bool    `json:"differentialConforms,omitempty"`
}

type reportTotals struct {
	Cases               int `json:"cases"`
	EngineMismatches    int `json:"engineMismatches"`
	ReferenceMismatches int `json:"referenceMismatches"`
	DifferentialMatches int `json:"differentialMatches"`
}

type phpstanOutput struct {
	Files map[string]struct {
		Messages []struct {
			Identifier string `json:"identifier"`
		} `json:"messages"`
	} `json:"files"`
	Errors []string `json:"errors"`
}

func main() {
	fixtures := flag.String("fixtures", "testdata/diagnostic-differential", "directory containing manifest.json and PHP fixtures")
	phpstanBin := flag.String("phpstan-bin", "phpstan", "PHPStan executable used for the reference run")
	engineOnly := flag.Bool("engine-only", false, "validate only the checked-in engine expectations")
	jsonOutput := flag.Bool("json", false, "write the machine-readable report to stdout")
	flag.Parse()

	report, err := runDifferential(*fixtures, *phpstanBin, *engineOnly)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if *jsonOutput {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(report); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	} else {
		printTextReport(report)
	}
	if report.Totals.EngineMismatches > 0 || report.Totals.ReferenceMismatches > 0 {
		os.Exit(1)
	}
}

func runDifferential(fixtures, phpstanBin string, engineOnly bool) (differentialReport, error) {
	manifest, err := loadManifest(filepath.Join(fixtures, "manifest.json"))
	if err != nil {
		return differentialReport{}, err
	}
	report := differentialReport{SchemaVersion: reportSchemaVersion, Engine: "go-php-parser", Totals: reportTotals{Cases: len(manifest.Cases)}}

	if !engineOnly {
		version, err := phpstanVersion(phpstanBin)
		if err != nil {
			return differentialReport{}, fmt.Errorf("reference analyser unavailable (use --engine-only for the local gate): %w", err)
		}
		report.Reference = &toolReport{Tool: manifest.Reference.Tool, Version: version, Level: manifest.Reference.Level}
	}

	for _, fixture := range manifest.Cases {
		path := filepath.Join(fixtures, fixture.File)
		actualEngine, err := runEngine(path, manifest.Reference.Level)
		if err != nil {
			return differentialReport{}, fmt.Errorf("case %s: %w", fixture.ID, err)
		}
		result := caseReport{
			ID:             fixture.ID,
			Capability:     fixture.Capability,
			File:           fixture.File,
			ExpectedEngine: sortedCopy(fixture.EngineCodes),
			ActualEngine:   actualEngine,
		}
		result.EngineMatches = equalStrings(result.ExpectedEngine, result.ActualEngine)
		if !result.EngineMatches {
			report.Totals.EngineMismatches++
		}

		if !engineOnly {
			actualReference, err := runPHPStan(phpstanBin, path, manifest.Reference.Level)
			if err != nil {
				return differentialReport{}, fmt.Errorf("case %s: %w", fixture.ID, err)
			}
			result.ExpectedReference = sortedCopy(fixture.PHPStanIdentifiers)
			result.ActualReference = actualReference
			matches := equalStrings(result.ExpectedReference, result.ActualReference)
			result.ReferenceMatches = &matches
			conforms := result.EngineMatches && matches
			result.DifferentialConforms = &conforms
			if !matches {
				report.Totals.ReferenceMismatches++
			}
			if conforms {
				report.Totals.DifferentialMatches++
			}
		}
		report.Cases = append(report.Cases, result)
	}
	return report, nil
}

func loadManifest(path string) (manifest, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return manifest{}, fmt.Errorf("read manifest: %w", err)
	}
	var result manifest
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil {
		return manifest{}, fmt.Errorf("decode manifest: %w", err)
	}
	if result.SchemaVersion != reportSchemaVersion {
		return manifest{}, fmt.Errorf("unsupported manifest schema version %d", result.SchemaVersion)
	}
	if result.Reference.Tool == "" || len(result.Cases) == 0 {
		return manifest{}, errors.New("manifest requires a reference tool and at least one case")
	}
	seen := make(map[string]struct{}, len(result.Cases))
	for i := range result.Cases {
		fixture := &result.Cases[i]
		if fixture.ID == "" || fixture.Capability == "" || fixture.File == "" {
			return manifest{}, fmt.Errorf("manifest case %d requires id, capability, and file", i)
		}
		if _, duplicate := seen[fixture.ID]; duplicate {
			return manifest{}, fmt.Errorf("duplicate manifest case id %q", fixture.ID)
		}
		seen[fixture.ID] = struct{}{}
		fixture.EngineCodes = sortedCopy(fixture.EngineCodes)
		fixture.PHPStanIdentifiers = sortedCopy(fixture.PHPStanIdentifiers)
	}
	return result, nil
}

func runEngine(path string, level int) ([]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read fixture: %w", err)
	}
	phpParser := parser.New(lexer.NewFileBytes(content), false)
	nodes := phpParser.Parse()
	if parseErrors := phpParser.Errors(); len(parseErrors) > 0 {
		return nil, fmt.Errorf("parse fixture: %s", strings.Join(parseErrors, "; "))
	}
	snapshot, err := analyse.NewSemanticSnapshot(map[string][]ast.Node{path: nodes}, nil)
	if err != nil {
		return nil, fmt.Errorf("build semantic snapshot: %w", err)
	}
	ctx := snapshot.NewAnalysisContext()
	ctx.AnalysisLevel = &level
	issues := analyse.RunAnalysisRulesWithContext(path, nodes, ctx)
	codes := make([]string, 0, len(issues))
	for _, issue := range issues {
		codes = append(codes, issue.Code)
	}
	sort.Strings(codes)
	return codes, nil
}

func phpstanVersion(binary string) (string, error) {
	output, err := exec.Command(binary, "--version").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s --version: %w: %s", binary, err, strings.TrimSpace(string(output)))
	}
	return strings.TrimSpace(string(output)), nil
}

func runPHPStan(binary, path string, level int) ([]string, error) {
	command := exec.Command(binary, "analyse", "--no-progress", "--error-format=json", fmt.Sprintf("--level=%d", level), path)
	output, err := command.Output()
	if err != nil {
		var exitError *exec.ExitError
		if !errors.As(err, &exitError) {
			return nil, fmt.Errorf("run PHPStan: %w", err)
		}
		if exitError.ExitCode() != 1 {
			return nil, fmt.Errorf("run PHPStan: %w: %s", err, strings.TrimSpace(string(exitError.Stderr)))
		}
	}
	return decodePHPStanIdentifiers(output)
}

func decodePHPStanIdentifiers(output []byte) ([]string, error) {
	var decoded phpstanOutput
	if err := json.Unmarshal(output, &decoded); err != nil {
		return nil, fmt.Errorf("decode PHPStan JSON: %w", err)
	}
	if len(decoded.Errors) > 0 {
		return nil, fmt.Errorf("PHPStan internal errors: %s", strings.Join(decoded.Errors, "; "))
	}
	identifiers := make([]string, 0)
	for _, file := range decoded.Files {
		for _, message := range file.Messages {
			identifier := message.Identifier
			if identifier == "" {
				identifier = "<unidentified>"
			}
			identifiers = append(identifiers, identifier)
		}
	}
	sort.Strings(identifiers)
	return identifiers, nil
}

func sortedCopy(values []string) []string {
	result := make([]string, len(values))
	copy(result, values)
	sort.Strings(result)
	return result
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func printTextReport(report differentialReport) {
	mode := "engine-only"
	if report.Reference != nil {
		mode = report.Reference.Version
	}
	fmt.Printf("Diagnostic differential: %d cases (%s)\n", report.Totals.Cases, mode)
	for _, fixture := range report.Cases {
		status := "PASS"
		if !fixture.EngineMatches || (fixture.ReferenceMatches != nil && !*fixture.ReferenceMatches) {
			status = "FAIL"
		}
		fmt.Printf("%s  %s  engine=%v", status, fixture.ID, fixture.ActualEngine)
		if fixture.ReferenceMatches != nil {
			fmt.Printf(" reference=%v", fixture.ActualReference)
		}
		fmt.Println()
	}
}
