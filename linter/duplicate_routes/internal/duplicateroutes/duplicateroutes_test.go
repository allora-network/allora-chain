package duplicateroutes

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestScanDetectsDuplicates(t *testing.T) {
	fsys := fstest.MapFS{
		"query.proto": {Data: joinLines(
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
		)},
	}

	findings, err := ScanFS(fsys, []string{"query.proto"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}

	finding := findings[0]
	if finding.File != "query.proto" {
		t.Fatalf("unexpected file: %s", finding.File)
	}
	if finding.Route != "/foo" {
		t.Fatalf("unexpected route: %s", finding.Route)
	}
	assertLines(t, finding.Lines, []int{4, 8})
}

func TestScanHandlesMultilineRoutes(t *testing.T) {
	fsys := fstest.MapFS{
		"query.proto": {Data: joinLines(
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
		)},
	}

	findings, err := ScanFS(fsys, []string{"query.proto"})
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
	fsys := fstest.MapFS{
		"query.proto": {Data: joinLines(
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
		)},
	}

	findings, err := ScanFS(fsys, []string{"query.proto"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(findings) != 0 {
		t.Fatalf("expected no findings, got %d", len(findings))
	}
}

func TestScanRecursesDirectories(t *testing.T) {
	fsys := fstest.MapFS{
		"a.proto": {Data: joinLines(
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
		)},
		"sub/b.proto": {Data: joinLines(
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
		)},
	}

	findings, err := ScanFS(fsys, nil)
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

	aProto := findingsByFile["a.proto"]
	assertLines(t, aProto.Lines, []int{4, 8})
	if aProto.Route != "/dup" {
		t.Fatalf("unexpected route for a.proto: %s", aProto.Route)
	}

	subB := findingsByFile["sub/b.proto"]
	assertLines(t, subB.Lines, []int{4, 8})
	if subB.Route != "/unique" {
		t.Fatalf("unexpected route for sub/b.proto: %s", subB.Route)
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

func joinLines(lines ...string) []byte {
	return []byte(strings.Join(lines, "\n") + "\n")
}
