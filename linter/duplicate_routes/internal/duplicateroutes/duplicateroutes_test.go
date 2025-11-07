package duplicateroutes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScanDetectsDuplicates(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "query.proto")
	writeFile(t, file, joinLines(
		`syntax = "proto3";`,
		`service Query {`,
		`  rpc Foo(Foo) returns (Foo) {`,
		`    option (google.api.http).get = "/foo";`,
		`  }`,
		``,
		`  rpc Bar(Bar) returns (Bar) {`,
		`    option (google.api.http).get = "/foo";`,
		`  }`,
		`}`,
	))

	findings, err := Scan([]string{file})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}

	finding := findings[0]
	if finding.File != file {
		t.Fatalf("unexpected file: %s", finding.File)
	}
	if finding.Route != "/foo" {
		t.Fatalf("unexpected route: %s", finding.Route)
	}
	assertLines(t, finding.Lines, []int{4, 8})
}

func TestScanHandlesMultilineRoutes(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "query.proto")
	writeFile(t, file, joinLines(
		`syntax = "proto3";`,
		`service Query {`,
		`  rpc Foo(Foo) returns (Foo) {`,
		`    option (google.api.http).get =`,
		`      "/foo/"`,
		`      "bar";`,
		`  }`,
		``,
		`  rpc Bar(Bar) returns (Bar) {`,
		`    option (google.api.http).get = "/foo/bar";`,
		`  }`,
		`}`,
	))

	findings, err := Scan([]string{file})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}

	finding := findings[0]
	if finding.Route != "/foo/bar" {
		t.Fatalf("unexpected route: %s", finding.Route)
	}
	assertLines(t, finding.Lines, []int{4, 10})
}

func TestScanIgnoresUniqueRoutes(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "query.proto")
	writeFile(t, file, joinLines(
		`syntax = "proto3";`,
		`service Query {`,
		`  rpc Foo(Foo) returns (Foo) {`,
		`    option (google.api.http).get = "/foo";`,
		`  }`,
		``,
		`  rpc Bar(Bar) returns (Bar) {`,
		`    option (google.api.http).get = "/bar";`,
		`  }`,
		`}`,
	))

	findings, err := Scan([]string{file})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(findings) != 0 {
		t.Fatalf("expected no findings, got %d", len(findings))
	}
}

func TestScanRecursesDirectories(t *testing.T) {
	root := t.TempDir()
	subDir := filepath.Join(root, "sub")
	if err := os.Mkdir(subDir, 0o755); err != nil {
		t.Fatalf("failed to create sub directory: %v", err)
	}

	file1 := filepath.Join(root, "a.proto")
	writeFile(t, file1, joinLines(
		`syntax = "proto3";`,
		`service Query {`,
		`  rpc Foo(Foo) returns (Foo) {`,
		`    option (google.api.http).get = "/dup";`,
		`  }`,
		``,
		`  rpc Bar(Bar) returns (Bar) {`,
		`    option (google.api.http).get = "/dup";`,
		`  }`,
		`}`,
	))

	file2 := filepath.Join(subDir, "b.proto")
	writeFile(t, file2, joinLines(
		`syntax = "proto3";`,
		`service Query {`,
		`  rpc Foo(Foo) returns (Foo) {`,
		`    option (google.api.http).get = "/unique";`,
		`  }`,
		``,
		`  rpc Bar(Bar) returns (Bar) {`,
		`    option (google.api.http).get = "/unique";`,
		`  }`,
		`}`,
	))

	findings, err := Scan([]string{root})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(findings) != 2 {
		t.Fatalf("expected 2 findings, got %d", len(findings))
	}

	findingsByFile := map[string]Finding{}
	for _, finding := range findings {
		findingsByFile[finding.File] = finding
	}

	assertLines(t, findingsByFile[file1].Lines, []int{4, 8})
	if findingsByFile[file1].Route != "/dup" {
		t.Fatalf("unexpected route for file1: %s", findingsByFile[file1].Route)
	}

	assertLines(t, findingsByFile[file2].Lines, []int{4, 8})
	if findingsByFile[file2].Route != "/unique" {
		t.Fatalf("unexpected route for file2: %s", findingsByFile[file2].Route)
	}
}

func assertLines(t *testing.T, got, want []int) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("unexpected line count: got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected lines: got %v want %v", got, want)
		}
	}
}

func joinLines(lines ...string) string {
	return strings.Join(lines, "\n") + "\n"
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("failed to write file %s: %v", path, err)
	}
}
