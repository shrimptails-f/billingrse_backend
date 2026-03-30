package nplusonecheck

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	analysistest.Run(
		t,
		analysistest.TestData(),
		newAnalyzer(),
		"nplusone_sql",
		"nplusone_gorm",
		"nplusone_helper",
		"nplusone_method",
		"nplusone_crosspkg_helper",
		"nplusone_crosspkg_method",
	)
}
