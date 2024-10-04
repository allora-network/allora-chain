package main

import (
	"flag"
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

var AnalyzerPlugin = map[string]*analysis.Analyzer{
	"maprange": Analyzer,
}

var Analyzer = &analysis.Analyzer{
	Name:             "maprange",
	Doc:              "check for range loops over maps",
	Run:              run,
	URL:              "",
	Flags:            flag.FlagSet{Usage: nil},
	RunDespiteErrors: false,
	Requires:         nil,
	ResultType:       nil,
	FactTypes:        nil,
}

func New(conf any) ([]*analysis.Analyzer, error) {
	return []*analysis.Analyzer{Analyzer}, nil
}

func run(pass *analysis.Pass) (interface{}, error) {
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			rangeStmt, ok := n.(*ast.RangeStmt)
			if !ok {
				return true
			}

			exprType := pass.TypesInfo.TypeOf(rangeStmt.X)
			if _, ok := exprType.Underlying().(*types.Map); ok {
				pass.Reportf(rangeStmt.Pos(), "range over map detected, which can be non-deterministic")
			}

			return true
		})
	}
	return nil, nil //nolint:nilnil
}
