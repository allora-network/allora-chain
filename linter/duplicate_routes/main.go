package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/allora-network/allora-chain/linter/duplicate_routes/internal/duplicateroutes"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [path ...]\n", os.Args[0])
		fmt.Fprintln(flag.CommandLine.Output(), "Scans protobuf files for duplicate google.api.http GET routes.")
	}
	flag.Parse()

	findings, err := duplicateroutes.Scan(flag.Args())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}

	if len(findings) == 0 {
		return
	}

	printFindings(findings)
	os.Exit(1)
}

func printFindings(findings []duplicateroutes.Finding) {
	grouped := make(map[string][]duplicateroutes.Finding)
	for _, finding := range findings {
		grouped[finding.File] = append(grouped[finding.File], finding)
	}

	files := make([]string, 0, len(grouped))
	for file := range grouped {
		files = append(files, file)
	}
	sort.Strings(files)

	for _, file := range files {
		fmt.Printf("%s:\n", file)
		routes := grouped[file]
		sort.Slice(routes, func(i, j int) bool {
			if routes[i].Route == routes[j].Route {
				return compareLines(routes[i].Lines, routes[j].Lines) < 0
			}
			return routes[i].Route < routes[j].Route
		})

		for _, finding := range routes {
			fmt.Printf("  duplicate route '%s' at lines %s\n", finding.Route, joinLines(finding.Lines))
		}
	}
}

func joinLines(lines []int) string {
	parts := make([]string, len(lines))
	for i, line := range lines {
		parts[i] = strconv.Itoa(line)
	}
	return strings.Join(parts, ", ")
}

func compareLines(a, b []int) int {
	min := len(a)
	if len(b) < min {
		min = len(b)
	}
	for i := 0; i < min; i++ {
		if a[i] != b[i] {
			return a[i] - b[i]
		}
	}
	return len(a) - len(b)
}
