package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/allora-network/allora-chain/x/utils/fn"
)

type ASTNode struct {
	Node           ast.Node
	Parent         *ASTNode
	Children       []*ASTNode
	DeferredCloses map[string]bool // Track deferred Close() calls by object name
}

var rootNode *ASTNode

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: go run main.go <directory>")
		os.Exit(1)
	}

	var hasErrors bool

	dir := os.Args[1]
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".go") {
			errs := processFile(path)
			if errs != nil {
				hasErrors = true
				errStrs := fn.Map(errs, func(err error) string { return err.Error() })
				fmt.Printf("Errors processing %s:\n", path)
				fmt.Println("  - " + strings.Join(errStrs, "\n  - "))
			}
		}
		return nil
	})

	if err != nil {
		hasErrors = true
		fmt.Printf("Error walking the path %s: %v\n", dir, err)
	}

	if hasErrors {
		os.Exit(2)
	}
}

func processFile(filename string) []error {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return []error{err}
	}

	rootNode = &ASTNode{Node: node, Parent: nil, Children: nil, DeferredCloses: make(map[string]bool)}
	currentNode := rootNode
	nodeStack := []*ASTNode{rootNode}

	ast.Inspect(node, func(n ast.Node) bool {
		if n == nil {
			if len(nodeStack) > 1 {
				nodeStack = nodeStack[:len(nodeStack)-1]
				currentNode = nodeStack[len(nodeStack)-1]
			}
			return false
		}
		if n != node {
			newNode := &ASTNode{Node: n, Parent: currentNode, Children: nil, DeferredCloses: make(map[string]bool)}
			currentNode.Children = append(currentNode.Children, newNode)
			currentNode = newNode
			nodeStack = append(nodeStack, currentNode)
		}
		return true
	})

	errs := checkNonDeferredClose(rootNode, fset)
	return errs
}

func checkNonDeferredClose(node *ASTNode, fset *token.FileSet) []error {
	var errs []error

	if call, ok := node.Node.(*ast.CallExpr); ok {
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
			if sel.Sel.Name == "Close" {
				objectName := getObjectName(sel.X)
				if isDeferredClose(node) {
					// Mark this object as having a deferred Close()
					markDeferredClose(node, objectName)
				} else {
					// Check if there's a preceding deferred Close() for this object
					if !hasDeferredClose(node, objectName) {
						pos := fset.Position(call.Pos())
						errs = append(errs, fmt.Errorf("%d: Close() call without preceding defer for `%s`", pos.Line, objectName))
					}
				}
			}
		}
	} else if deferStmt, ok := node.Node.(*ast.DeferStmt); ok {
		if deferStmt.Call != nil {
			if sel, ok := deferStmt.Call.Fun.(*ast.SelectorExpr); ok {
				if sel.Sel.Name == "Close" {
					objectName := getObjectName(sel.X)
					markDeferredClose(node, objectName)
				}
			}
		}
	}

	for _, child := range node.Children {
		childErrs := checkNonDeferredClose(child, fset)
		errs = append(errs, childErrs...)
	}
	return errs
}

func getObjectName(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return getObjectName(e.X) + "." + e.Sel.Name
	default:
		return "unknown"
	}
}

// `markDeferredClose` marks an object as having a deferred Close() call
// within the scope of its enclosing function.
//
// This function traverses up the AST from the given node, marking each node
// as having a deferred Close() for the specified object. It stops when it
// reaches a function declaration or literal, as defer statements are only
// meaningful within the context of their enclosing function.
func markDeferredClose(node *ASTNode, objectName string) {
	current := node
	for current != nil && !isFunc(current) {
		current.DeferredCloses[objectName] = true
		current = current.Parent
	}
}

func isFunc(node *ASTNode) bool {
	switch node.Node.(type) {
	case *ast.FuncDecl, *ast.FuncLit:
		return true
	}
	return false
}

func hasDeferredClose(node *ASTNode, objectName string) bool {
	current := node
	for current != nil {
		if current.DeferredCloses[objectName] {
			return true
		}
		current = current.Parent
	}
	return false
}

func isDeferredClose(node *ASTNode) bool {
	current := node.Parent
	for current != nil {
		switch n := current.Node.(type) {
		case *ast.DeferStmt:
			return true
		case *ast.FuncLit:
			if isDeferredFuncLit(n) {
				return true
			}
		case *ast.IfStmt, *ast.ForStmt, *ast.SwitchStmt, *ast.SelectStmt, *ast.BlockStmt:
			// Continue traversing up for control structures
		case *ast.FuncDecl, *ast.File:
			// We've reached the top of the function or file without finding a defer
			return false
		}
		current = current.Parent
	}
	return false
}

// `isDeferredFuncLit` checks if a given function literal is directly deferred.
// It traverses up the AST from the function literal node, looking for a defer statement
// that immediately wraps this function literal.
//
// The function works as follows:
// 1. It first finds the ASTNode corresponding to the given function literal.
// 2. If the node is not found in the AST, it returns false.
// 3. It then traverses up the parent chain of the node.
// 4. If it encounters a defer statement that calls this function literal, it returns true.
// 5. If it reaches the top of the AST without finding a defer statement, it returns false.
func isDeferredFuncLit(funcLit *ast.FuncLit) bool {
	current := findNodeByType(rootNode, funcLit)
	if current == nil {
		return false
	}
	for current.Parent != nil {
		switch n := current.Parent.Node.(type) {
		case *ast.DeferStmt:
			return n.Call.Fun == funcLit
		}
		current = current.Parent
	}
	return false
}

// `findNodeByType` searches for a node of a specific type in the AST.
// It performs a depth-first search starting from the given root node.
// If a node matching the target type is found, it returns the ASTNode containing that node.
// If no matching node is found, it returns nil.
func findNodeByType(root *ASTNode, target ast.Node) *ASTNode {
	if root.Node == target {
		return root
	}
	for _, child := range root.Children {
		if found := findNodeByType(child, target); found != nil {
			return found
		}
	}
	return nil
}
