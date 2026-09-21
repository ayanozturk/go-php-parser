package phpstubs

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"strings"
)

//go:embed 8.2/*.php 8.3/*.php 8.4/*.php 8.5/*.php shared/*.php
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

// cleanStubName strips a trailing .php and rejects empty or path-like names so
// Read/ReadShared cannot escape their version or shared directories via "..".
func cleanStubName(name string) string {
	name = strings.TrimSuffix(strings.TrimSpace(name), ".php")
	if name == "" || name == "." || name == ".." {
		return ""
	}
	if strings.ContainsAny(name, `/\`) || name != path.Base(name) {
		return ""
	}
	return name
}

func stubBaseNames(fsys fs.FS, dir string) []string {
	entries, err := fs.ReadDir(fsys, dir)
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

// Names returns bundled stub files for a PHP version, without the .php suffix.
func Names(version string) []string {
	return stubBaseNames(content, NormalizePHPVersion(version))
}

// Read returns the stub source for an extension name such as "Core" or "SPL".
func Read(version, name string) ([]byte, error) {
	version = NormalizePHPVersion(version)
	name = cleanStubName(name)
	if name == "" {
		return nil, fmt.Errorf("phpstubs: invalid stub name")
	}
	return content.ReadFile(path.Join(version, name+".php"))
}

// FileName is the virtual project-index path for a bundled stub.
func FileName(version, name string) string {
	version = NormalizePHPVersion(version)
	name = cleanStubName(name)
	return "phpstub:" + version + "/" + name + ".php"
}

// SharedNames returns version-independent stub files such as Standard.
func SharedNames() []string {
	return stubBaseNames(content, "shared")
}

// ReadShared returns a version-independent stub source file.
func ReadShared(name string) ([]byte, error) {
	name = cleanStubName(name)
	if name == "" {
		return nil, fmt.Errorf("phpstubs: invalid stub name")
	}
	return content.ReadFile(path.Join("shared", name+".php"))
}

// SharedFileName is the virtual project-index path for a shared stub.
func SharedFileName(name string) string {
	name = cleanStubName(name)
	return "phpstub:shared/" + name + ".php"
}
