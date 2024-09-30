package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

type ASTNode struct {
	Node     ast.Node
	Parent   *ASTNode
	Children []*ASTNode
}

var rootNode *ASTNode

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: go run main.go <directory>")
		os.Exit(1)
	}

	dir := os.Args[1]
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".go") {
			err := processFile(path)
			if err != nil {
				fmt.Printf("Error processing %s: %v\n", path, err)
			}
		}
		return nil
	})

	if err != nil {
		fmt.Printf("Error walking the path %s: %v\n", dir, err)
	}
}

func processFile(filename string) error {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	rootNode = &ASTNode{Node: node}
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
			newNode := &ASTNode{Node: n, Parent: currentNode}
			currentNode.Children = append(currentNode.Children, newNode)
			currentNode = newNode
			nodeStack = append(nodeStack, currentNode)
		}
		return true
	})

	checkNonDeferredClose(rootNode, fset)

	return nil
}

func checkNonDeferredClose(node *ASTNode, fset *token.FileSet) {
	if call, ok := node.Node.(*ast.CallExpr); ok {
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
			if sel.Sel.Name == "Close" {
				if !isDeferredClose(node) {
					pos := fset.Position(call.Pos())
					fmt.Printf("%s:%d: Close() call without defer\n", pos.Filename, pos.Line)
				}
			}
		}
	}

	for _, child := range node.Children {
		checkNonDeferredClose(child, fset)
	}
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
