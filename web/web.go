// web/web.go

package web

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"snap-ci/storage"
	"strings"
)

var funcMap = template.FuncMap{
	"lower": strings.ToLower,
}

func runHistoryHandler(w http.ResponseWriter, r *http.Request) {
	runs, err := storage.GetRecentRuns(10) // Get the 10 most recent runs
	if err != nil {
		log.Printf("Error fetching recent runs: %v", err)
		http.Error(w, "Failed to load run history", http.StatusInternalServerError)
		return
	}

	tmpl, err := template.New("run_history.html").Funcs(funcMap).ParseFiles("/home/chinmay/Documents/snap-ci/web/templates/run_history.html")
	if err != nil {
		log.Printf("Error parsing template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, runs); err != nil {
		log.Printf("Error executing template: %v", err)
	}
}

func runDetailsHandler(w http.ResponseWriter, r *http.Request) {
	runIDStr := r.URL.Path[len("/runs/"):] // Extract run ID from path
	runID := runIDStr                      // Assuming run ID is a string

	run, err := storage.GetRun(runID)
	if err != nil {
		log.Printf("Error fetching run %s: %v", runID, err)
		http.NotFound(w, r)
		return
	}

	tmpl, err := template.New("run_details.html").Funcs(funcMap).ParseFiles("/home/chinmay/Documents/snap-ci/web/templates/run_details.html")
	if err != nil {
		log.Printf("Error parsing template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, run); err != nil {
		log.Printf("Error executing template: %v", err)
	}
}

func StartWebServer() error {
	http.HandleFunc("/", runHistoryHandler)
	http.HandleFunc("/runs/", runDetailsHandler) // Handle requests like /runs/20250603140000

	port := ":8081" // Choose a different port than the webhook listener
	fmt.Printf("Web dashboard listening on port %s...\n", port)
	err := http.ListenAndServe(port, nil)
	if err != nil {
		return fmt.Errorf("failed to start web server: %w", err)
	}
	return nil
}
