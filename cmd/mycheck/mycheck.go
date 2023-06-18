// Модуль osexitcheckanalyzer осуществляет статическую проверку на вызов функции os.Exit
package osexitcheckanalyzer

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// OsExitCheckAnalyzer переменная для экспорта
var OsExitCheckAnalyzer = &analysis.Analyzer{
	Name: "osexitcheck",
	Doc:  "check for os.Exit usage in main",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	for _, file := range pass.Files {
		// функцией ast.Inspect проходим по всем узлам AST
		ast.Inspect(file, func(n ast.Node) bool {
			if c, ok := n.(*ast.CallExpr); ok {
				if s, ok := c.Fun.(*ast.SelectorExpr); ok {
					// только функции Println
					if s.Sel.Name == "Exit" {
						pass.Reportf(s.Pos(), "used Exit")
					}
				}
			}
			return true
		})
	}
	return nil, nil
}
