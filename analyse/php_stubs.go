package analyse

import (
	"sort"
	"sync"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/lexer"
	"github.com/ayanozturk/go-php-parser/parser"
	"github.com/ayanozturk/go-php-parser/phpstubs"
)

var parsedPHPStubs sync.Map // version -> map[string][]ast.Node

func parsedStubsForVersion(version string) map[string][]ast.Node {
	version = phpstubs.NormalizePHPVersion(version)
	if cached, ok := parsedPHPStubs.Load(version); ok {
		return cached.(map[string][]ast.Node)
	}
	parsed := make(map[string][]ast.Node)
	for _, name := range phpstubs.Names(version) {
		src, err := phpstubs.Read(version, name)
		if err != nil {
			continue
		}
		p := parser.New(lexer.New(string(src)), false)
		nodes := p.Parse()
		parsed[phpstubs.FileName(version, name)] = nodes
	}
	actual, _ := parsedPHPStubs.LoadOrStore(version, parsed)
	return actual.(map[string][]ast.Node)
}

func (idx *ProjectIndex) indexPHPStubs(version string) {
	parsed := parsedStubsForVersion(version)
	filenames := make([]string, 0, len(parsed))
	for filename := range parsed {
		filenames = append(filenames, filename)
	}
	sort.Strings(filenames)
	for _, filename := range filenames {
		nodes := parsed[filename]
		ft := CollectFileTypeContext(nodes)
		idx.FileTypes[filename] = ft
		idx.indexNodes(filename, nodes, ft, "")
	}
}
