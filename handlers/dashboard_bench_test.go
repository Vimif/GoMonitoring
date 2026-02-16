package handlers

import (
	"html/template"
	"os"
	"testing"
)

func init() {
	// Change to root directory to find templates/
	// This assumes tests are run from handlers/ directory
	os.Chdir("..")
	wd, _ := os.Getwd()
	println("Current Working Directory:", wd)
}

func BenchmarkGetDashboardTemplate(b *testing.B) {
	// Ensure template is initialized first to avoid race/error during benchmark setup if any
	_, err := getDashboardTemplate()
	if err != nil {
		b.Fatal("Failed to initialize template:", err)
	}

	// Benchmark cached access
	b.Run("Cached", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := getDashboardTemplate()
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	// Benchmark uncached parsing (simulating old behavior)
	b.Run("Uncached", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			// We must recreate the template object every time to simulate full parsing
			_, err := template.New("base.html").Funcs(dashboardFuncs).ParseFiles(
				"templates/layout/base.html",
				"templates/dashboard.html",
			)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
