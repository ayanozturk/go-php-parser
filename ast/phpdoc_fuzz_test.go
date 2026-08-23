package ast

import "testing"

const maxPHPDocFuzzBytes = 64 << 10

func FuzzParsePHPDoc(f *testing.F) {
	seeds := []string{
		"",
		"/** */",
		"/** @param array<string, list<int>> $items */",
		"/** @return (Alpha&Beta)|null */",
		"/** @template T of Entity */",
		"/** @extends Repository<Map<string, list<T>>> */",
		"/** @param array{open: string, nested: array<int, string>} $shape */",
		string([]byte{'/', '*', '*', ' ', 0xff, 0xfe, 0x00, ' ', '*', '/'}),
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, raw string) {
		if len(raw) > maxPHPDocFuzzBytes {
			t.Skip()
		}

		doc := ParsePHPDoc(raw)
		if doc == nil || doc.RawContent != raw {
			t.Fatalf("PHPDoc parsing corrupted raw content")
		}
	})
}
