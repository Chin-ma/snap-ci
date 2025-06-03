package web

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	"snap-ci/storage"
)

// handleJobList displays a list of recent pipeline runs
func handleJobList(w http.ResponseWriter, r *http.Request) {
	runs, err := storage.GetRecentRuns(10)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching recent runs: %v", err), http.StatusInternalServerError)
		return
	}

	// Load the template
	tmpl, err := template.ParseFiles("web/templates/job_list.html")
	if err != nil {
		http.Error(w, fmt.Sprintf("Error loading template: %v", err), http.StatusInternalServerError)
		return
	}

	// Execute the template with data
	if err := tmpl.Execute(w, map[string]interface{}{"Runs": runs}); err != nil {
		http.Error(w, fmt.Sprintf("Error executing template: %v", err), http.StatusInternalServerError)
		return
	}
}

// handleJobDetails displays details for a specific job run
func handleJobDetails(w http.ResponseWriter, r *http.Request) {
	// Extract job ID from URL
	jobIDStr := r.URL.Path[len("/job/"):]
	jobID, err := strconv.Atoi(jobIDStr)
	if err != nil {
		http.Error(w, "Invalid Job ID", http.StatusBadRequest)
		return
	}

	run, err := storage.GetRun(strconv.Itoa(jobID)) // Convert jobID (int) to string for GetRun
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching job details: %v", err), http.StatusInternalServerError)
		return
	}

	// Load the template
	tmpl, err := template.ParseFiles("web/templates/job_details.html")
	if err != nil {
		http.Error(w, fmt.Sprintf("Error loading template: %v", err), http.StatusInternalServerError)
		return
	}

	// Execute the template with data
	if err := tmpl.Execute(w, map[string]interface{}{"Run": run}); err != nil {
		http.Error(w, fmt.Sprintf("Error executing template: %v", err), http.StatusInternalServerError)
		return
	}
}
