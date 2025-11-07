package duplicateroutes

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	optionPattern = regexp.MustCompile(`option\s*\(google\.api\.http\)\.get`)
	stringPattern = regexp.MustCompile(`"([^"\\]*(?:\\.[^"\\]*)*)"`)
)

// Finding represents a duplicated route within a protobuf file.
type Finding struct {
	File  string
	Route string
	Lines []int
}

// FileSystem encapsulates the filesystem interactions required by the scanner.
type FileSystem interface {
	Stat(string) (fs.FileInfo, error)
	WalkDir(string, fs.WalkDirFunc) error
	ReadFile(string) ([]byte, error)
}

type osFS struct{}

func (osFS) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}

func (osFS) WalkDir(root string, fn fs.WalkDirFunc) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		return fn(path, d, err)
	})
}

func (osFS) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

// Scan walks the provided paths, recursively searching for duplicate google.api.http routes.
// When paths is empty, the current working directory is used.
func Scan(paths []string) ([]Finding, error) {
	return scanWithFilesystem(osFS{}, paths)
}

// ScanFS performs the scan against an arbitrary fs.FS implementation.
// When roots is empty, the scan starts from ".".
func ScanFS(fsys fs.FS, roots []string) ([]Finding, error) {
	return scanWithFilesystem(fsAdapter{fsys: fsys}, roots)
}

func scanWithFilesystem(fsys FileSystem, paths []string) ([]Finding, error) {
	if len(paths) == 0 {
		paths = []string{"."}
	}

	aggregated := make(map[string]map[string][]int)
	for _, root := range paths {
		if err := processPath(fsys, root, aggregated); err != nil {
			return nil, err
		}
	}

	return finalizeFindings(aggregated), nil
}

func processPath(fsys FileSystem, root string, aggregated map[string]map[string][]int) error {
	if root == "" {
		root = "."
	}

	info, err := fsys.Stat(root)
	if err != nil {
		return err
	}

	if !info.IsDir() {
		return processFile(fsys, root, aggregated)
	}

	return fsys.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".proto" {
			return nil
		}
		return processFile(fsys, path, aggregated)
	})
}

func processFile(fsys FileSystem, path string, aggregated map[string]map[string][]int) error {
	if filepath.Ext(path) != ".proto" {
		return nil
	}

	duplicates, err := findDuplicates(fsys, path)
	if err != nil {
		return err
	}
	if len(duplicates) == 0 {
		return nil
	}

	if _, ok := aggregated[path]; !ok {
		aggregated[path] = make(map[string][]int)
	}

	for route, lines := range duplicates {
		aggregated[path][route] = append(aggregated[path][route], lines...)
	}

	return nil
}

func findDuplicates(fsys FileSystem, path string) (map[string][]int, error) {
	data, err := fsys.ReadFile(path)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	occur := make(map[string][]int)

	for index := 0; index < len(lines); index++ {
		line := lines[index]
		if !optionPattern.MatchString(line) {
			continue
		}

		parts, last := collectRouteParts(lines, index)
		if len(parts) == 0 {
			return nil, fmt.Errorf("no route string literal found in %s at line %d", path, index+1)
		}

		route, err := buildRoute(parts)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}

		occur[route] = append(occur[route], index+1)
		index = last
	}

	duplicates := make(map[string][]int)
	for route, positions := range occur {
		if len(positions) > 1 {
			linesCopy := append([]int(nil), positions...)
			sort.Ints(linesCopy)
			duplicates[route] = linesCopy
		}
	}
	return duplicates, nil
}

type fsAdapter struct {
	fsys fs.FS
}

func (a fsAdapter) Stat(name string) (fs.FileInfo, error) {
	return fs.Stat(a.fsys, name)
}

func (a fsAdapter) WalkDir(root string, fn fs.WalkDirFunc) error {
	return fs.WalkDir(a.fsys, root, fn)
}

func (a fsAdapter) ReadFile(name string) ([]byte, error) {
	return fs.ReadFile(a.fsys, name)
}

func collectRouteParts(lines []string, start int) ([]string, int) {
	var parts []string
	i := start

	for {
		matches := stringPattern.FindAllStringSubmatch(lines[i], -1)
		for _, match := range matches {
			parts = append(parts, match[1])
		}
		if strings.Contains(lines[i], ";") || i+1 >= len(lines) {
			break
		}
		i++
	}

	return parts, i
}

func buildRoute(parts []string) (string, error) {
	var builder strings.Builder
	for _, raw := range parts {
		segment, err := strconv.Unquote(`"` + raw + `"`)
		if err != nil {
			return "", fmt.Errorf("invalid escape sequence in route segment: %w", err)
		}
		builder.WriteString(segment)
	}

	route := builder.String()
	if strings.TrimSpace(route) == "" {
		return "", errors.New("empty route detected")
	}
	return route, nil
}

func finalizeFindings(aggregated map[string]map[string][]int) []Finding {
	var findings []Finding
	for file, routes := range aggregated {
		for route, lines := range routes {
			if len(lines) < 2 {
				continue
			}
			linesCopy := append([]int(nil), lines...)
			sort.Ints(linesCopy)
			findings = append(findings, Finding{
				File:  file,
				Route: route,
				Lines: linesCopy,
			})
		}
	}

	sort.Slice(findings, func(i, j int) bool {
		if findings[i].File == findings[j].File {
			return findings[i].Route < findings[j].Route
		}
		return findings[i].File < findings[j].File
	})
	return findings
}
