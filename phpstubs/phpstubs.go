package phpstubs

import (
	"embed"
	"io/fs"
	"path"
	"strings"
)

//go:embed 8.2/*.php 8.3/*.php 8.4/*.php 8.5/*.php
var content embed.FS

const DefaultPHPVersion = "8.3"

var supportedPHPVersions = []string{"8.2", "8.3", "8.4", "8.5"}

// NormalizePHPVersion maps a configured version onto a bundled stub set.
func NormalizePHPVersion(version string) string {
	version = strings.TrimSpace(version)
	if version == "" {
		return DefaultPHPVersion
	}
	majorMinor := version
	if parts := strings.Split(version, "."); len(parts) >= 2 {
		majorMinor = parts[0] + "." + parts[1]
	}
	for _, supported := range supportedPHPVersions {
		if majorMinor == supported {
			return supported
		}
	}
	return DefaultPHPVersion
}

// Names returns bundled stub files for a PHP version, without the .php suffix.
func Names(version string) []string {
	version = NormalizePHPVersion(version)
	entries, err := fs.ReadDir(content, version)
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".php") {
			continue
		}
		names = append(names, strings.TrimSuffix(entry.Name(), ".php"))
	}
	return names
}

// Read returns the stub source for an extension name such as "Core" or "SPL".
func Read(version, name string) ([]byte, error) {
	version = NormalizePHPVersion(version)
	name = strings.TrimSuffix(strings.TrimSpace(name), ".php")
	return content.ReadFile(path.Join(version, name+".php"))
}

// FileName is the virtual project-index path for a bundled stub.
func FileName(version, name string) string {
	version = NormalizePHPVersion(version)
	name = strings.TrimSuffix(strings.TrimSpace(name), ".php")
	return "phpstub:" + version + "/" + name + ".php"
}
