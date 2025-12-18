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

// Scan walks the provided paths, recursively searching for duplicate google.api.http routes.
// When paths is empty, the current working directory is used.
func Scan(paths []string) ([]Finding, error) {
	roots, err := makeHostRoots(paths)
	if err != nil {
		return nil, err
	}
	return scanWithFilesystem(os.DirFS("/"), roots)
}

// ScanFS performs the scan against an arbitrary fs.FS implementation.
// When roots is empty, the scan starts from ".".
func ScanFS(fsys fs.FS, roots []string) ([]Finding, error) {
	return scanWithFilesystem(fsys, makeGenericRoots(roots))
}

// scanWithFilesystem executes the scan for all roots against the provided filesystem.
func scanWithFilesystem(fsys fs.FS, roots []scanRoot) ([]Finding, error) {
	if len(roots) == 0 {
		roots = []scanRoot{{fsPath: ".", resolve: identityResolver}}
	}

	aggregated := make(map[string]map[string][]int)
	for _, root := range roots {
		if err := processPath(fsys, root, aggregated); err != nil {
			return nil, err
		}
	}

	return finalizeFindings(aggregated), nil
}

// processPath walks a root path and feeds individual proto files into the aggregator.
func processPath(fsys fs.FS, root scanRoot, aggregated map[string]map[string][]int) error {
	fsPath := root.fsPath
	if fsPath == "" {
		fsPath = "."
	}

	info, err := fs.Stat(fsys, fsPath)
	if err != nil {
		return err
	}

	if !info.IsDir() {
		return processFile(fsys, fsPath, root.resolve, aggregated)
	}

	return fs.WalkDir(fsys, fsPath, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".proto" {
			return nil
		}
		return processFile(fsys, path, root.resolve, aggregated)
	})
}

// processFile records duplicate routes found inside a single proto file.
func processFile(fsys fs.FS, path string, resolve pathResolver, aggregated map[string]map[string][]int) error {
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

	displayPath := resolve(path)
	if _, ok := aggregated[displayPath]; !ok {
		aggregated[displayPath] = make(map[string][]int)
	}

	for route, lines := range duplicates {
		aggregated[displayPath][route] = append(aggregated[displayPath][route], lines...)
	}

	return nil
}

// findDuplicates detects repeated route literals within a proto file.
func findDuplicates(fsys fs.FS, path string) (map[string][]int, error) {
	data, err := fs.ReadFile(fsys, path)
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

type pathResolver func(string) string

type scanRoot struct {
	fsPath  string
	resolve pathResolver
}

// identityResolver returns paths without modification.
func identityResolver(path string) string {
	return path
}

// makeGenericRoots prepares scan roots for arbitrary filesystem implementations.
func makeGenericRoots(roots []string) []scanRoot {
	if len(roots) == 0 {
		roots = []string{"."}
	}
	result := make([]scanRoot, 0, len(roots))
	for _, root := range roots {
		if root == "" {
			root = "."
		}
		result = append(result, scanRoot{
			fsPath:  root,
			resolve: identityResolver,
		})
	}
	return result
}

// makeHostRoots prepares scan roots sourced from the host filesystem.
func makeHostRoots(paths []string) ([]scanRoot, error) {
	if len(paths) == 0 {
		paths = []string{"."}
	}
	result := make([]scanRoot, 0, len(paths))
	for _, input := range paths {
		if input == "" {
			input = "."
		}
		abs, err := filepath.Abs(input)
		if err != nil {
			return nil, err
		}
		fsPath := hostToFSPath(abs)
		resolver := hostPathResolver(fsPath, filepath.Clean(input))
		result = append(result, scanRoot{
			fsPath:  fsPath,
			resolve: resolver,
		})
	}
	return result, nil
}

// hostToFSPath converts an absolute host path into the fs.FS namespace.
func hostToFSPath(abs string) string {
	cleaned := filepath.Clean(abs)
	if cleaned == string(os.PathSeparator) {
		return "."
	}
	fsPath := strings.TrimPrefix(filepath.ToSlash(cleaned), "/")
	if fsPath == "" {
		return "."
	}
	return fsPath
}

// hostPathResolver builds a resolver that rewrites fs paths back to display paths.
func hostPathResolver(root, displayRoot string) pathResolver {
	cleanDisplay := filepath.Clean(displayRoot)
	if cleanDisplay == "" {
		cleanDisplay = "."
	}
	return func(full string) string {
		rel := relativeFromRoot(root, full)
		if rel == "" {
			return cleanDisplay
		}
		return filepath.Join(cleanDisplay, filepath.FromSlash(rel))
	}
}

// relativeFromRoot trims the shared root prefix from a full path.
func relativeFromRoot(root, full string) string {
	if root == "" || root == "." {
		return full
	}
	if full == root {
		return ""
	}
	prefix := root + "/"
	if strings.HasPrefix(full, prefix) {
		return strings.TrimPrefix(full, prefix)
	}
	return full
}

// collectRouteParts extracts contiguous string literal fragments starting at a line index.
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

// buildRoute stitches route fragments while honoring escape sequences.
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

// finalizeFindings converts the aggregated map into a sorted slice of findings.
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
