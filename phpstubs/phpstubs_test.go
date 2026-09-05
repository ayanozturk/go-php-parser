package phpstubs

import "testing"

func TestNormalizePHPVersionFallsBackToDefault(t *testing.T) {
	if NormalizePHPVersion("") != DefaultPHPVersion {
		t.Fatalf("expected default %s", DefaultPHPVersion)
	}
	if NormalizePHPVersion("8.4.12") != "8.4" {
		t.Fatalf("expected 8.4, got %s", NormalizePHPVersion("8.4.12"))
	}
	if NormalizePHPVersion("7.4") != DefaultPHPVersion {
		t.Fatalf("unsupported versions should fall back to %s", DefaultPHPVersion)
	}
}

func TestBundledStubsExistPerSupportedVersion(t *testing.T) {
	for _, version := range supportedPHPVersions {
		names := Names(version)
		if len(names) == 0 {
			t.Fatalf("expected bundled stubs for PHP %s", version)
		}
		for _, name := range []string{"Core", "SPL"} {
			if _, err := Read(version, name); err != nil {
				t.Fatalf("read %s %s: %v", version, name, err)
			}
		}
	}
}
