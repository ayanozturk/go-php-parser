package main

import "testing"

func TestParseMagoCountOutputSumsSeverities(t *testing.T) {
	got, err := parseMagoCountOutput("error: 1929\nwarning: 292\nnote: 179\nhelp: 4\n")
	if err != nil {
		t.Fatalf("parse mago count: %v", err)
	}
	if got != 2404 {
		t.Fatalf("count = %d, want 2404", got)
	}
}

func TestParseMagoCountOutputRejectsEmpty(t *testing.T) {
	if _, err := parseMagoCountOutput("INFO starting\n"); err == nil {
		t.Fatal("expected empty count output to fail")
	}
}

func TestMatchPathGlobDoubleStarAndPrefix(t *testing.T) {
	tests := []struct {
		pattern string
		path    string
		want    bool
	}{
		{pattern: "splitter/vendor", path: "splitter/vendor/autoload.php", want: true},
		{pattern: "*/packages/**/vendor/*", path: "nested/packages/foo/vendor/autoload.php", want: true},
		{pattern: "*/packages/**/vendor/*", path: "packages/foo/src/File.php", want: false},
		{pattern: "**/tests/**", path: "packages/vec/tests/unit/ZipTest.php", want: true},
		{pattern: "src/js", path: "src/keep.php", want: false},
	}
	for _, test := range tests {
		if got := benchmarkPathExcluded(test.path, []string{test.pattern}); got != test.want {
			t.Errorf("benchmarkPathExcluded(%q, %q) = %v, want %v", test.path, test.pattern, got, test.want)
		}
	}
}
