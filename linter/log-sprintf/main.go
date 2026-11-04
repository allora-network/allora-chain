package main

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/singlechecker"
)

var Analyzer = &analysis.Analyzer{ //nolint:exhaustruct
	Name: "log_sprintf",
	Doc:  "checks for inefficient use of fmt.Sprintf inside logging functions",
	Run:  run,
}

var loggingFuncs = map[string]bool{
	"Debug": true,
	"Info":  true,
	"Warn":  true,
	"Error": true,
	"Fatal": true,
}

func run(pass *analysis.Pass) (interface{}, error) {
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			callExpr, ok := n.(*ast.CallExpr)
			if !ok || len(callExpr.Args) == 0 {
				return true
			}

			// Check if the function being called is a SelectorExpr (e.g., logger.Debug)
			selExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}

			// Match known logging method names (regardless of object)
			if !loggingFuncs[selExpr.Sel.Name] {
				return true
			}

			// Check if first argument is fmt.Sprintf(...)
			firstArg, ok := callExpr.Args[0].(*ast.CallExpr)
			if !ok {
				return true
			}

			innerSel, ok := firstArg.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}

			pkgIdent, ok := innerSel.X.(*ast.Ident)
			if !ok || pkgIdent.Name != "fmt" || innerSel.Sel.Name != "Sprintf" {
				return true
			}

			pass.Reportf(firstArg.Pos(), "avoid using fmt.Sprintf inside log.%s: use logger formatting instead", selExpr.Sel.Name)
			return true
		})
	}
	return nil, nil //nolint:nilnil
}

func main() {
	singlechecker.Main(Analyzer)
}
