package main

import (
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"

	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/structtag"

	"honnef.co/go/tools/staticcheck"

	osexitcheckanalyzer "github.com/jon69/shorturl/cmd/mycheck"
)

func main() {
	// определяем map подключаемых правил
	checks := map[string]bool{
		"SA":     true,
		"S1006":  true,
		"S1017":  true,
		"ST1003": true,
		"QF1001": true,
	}
	var mychecks []*analysis.Analyzer

	mychecks = append(mychecks, printf.Analyzer)
	mychecks = append(mychecks, shadow.Analyzer)
	mychecks = append(mychecks, structtag.Analyzer)
	mychecks = append(mychecks, osexitcheckanalyzer.OsExitCheckAnalyzer)

	for _, v := range staticcheck.Analyzers {
		// добавляем в массив нужные проверки
		for checkName := range checks {
			if strings.HasPrefix(v.Analyzer.Name, checkName) {
				mychecks = append(mychecks, v.Analyzer)
			}
		}
	}
	multichecker.Main(
		mychecks...,
	)
}
