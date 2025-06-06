// web/web.go

package web

import (
	"embed" // Import the embed package
	"fmt"
	"html/template"
	"log"
	"net/http"
	"snap-ci/git"
	"snap-ci/storage"
	"strings"
)

var funcMap = template.FuncMap{
	"lower": strings.ToLower,
}

//go:embed templates/*.html
var templatesFs embed.FS

// Global template variable for simpler template parsing
// Parse templates from the embedded file system
var templates = template.Must(template.New("").Funcs(funcMap).ParseFS(templatesFs, "templates/*.html"))

func runHistoryHandler(w http.ResponseWriter, r *http.Request) {
	runs, err := storage.GetRecentRuns(10) // Get the 10 most recent runs
	if err != nil {
		log.Printf("Error fetching recent runs: %v", err)
		http.Error(w, "Failed to load run history", http.StatusInternalServerError)
		return
	}

	// Remove commented-out ParseFiles lines as they are no longer needed with embed
	// tmpl, err := template.New("run_history.html").Funcs(funcMap).ParseFiles("/home/chinmay/Documents/snap-ci/web/templates/run_history.html")
	// if err != nil {
	//  log.Printf("Error parsing template: %v", err)
	//  http.Error(w, "Internal server error", http.StatusInternalServerError)
	//  return
	// }

	// <--- FIX 2: Use ExecuteTemplate to specify which template from the collection to execute
	if err := templates.ExecuteTemplate(w, "run_history.html", runs); err != nil {
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

	// Remove commented-out ParseFiles lines as they are no longer needed with embed
	// tmpl, err := template.New("run_details.html").Funcs(funcMap).ParseFiles("/home/chinmay/Documents/snap-ci/web/templates/run_details.html")
	// if err != nil {
	//  log.Printf("Error parsing template: %v", err)
	//  http.Error(w, "Internal server error", http.StatusInternalServerError)
	//  return
	// }

	// <--- FIX 3: Use ExecuteTemplate to specify which template from the collection to execute
	if err := templates.ExecuteTemplate(w, "run_details.html", run); err != nil {
		log.Printf("Error executing template: %v", err)
	}
}

func setupWebhookHandler(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Message string
		Error   string
	}{}

	if r.Method == http.MethodPost {
		repo := r.FormValue("repo")
		token := r.FormValue("token")

		if repo == "" || token == "" {
			data.Error = "Repository and Token are required." // Changed from data.Message to data.Error for consistency
		} else {
			log.Printf("Attempting to set up webhook for %s via Web UI...", repo)
			if err := git.SetupGitHubWebhook(repo, token); err != nil {
				data.Error = fmt.Sprintf("Failed to set up GitHub webhook: %v", err)
				log.Printf("Error setting up webhook via Web UI for %s: %v", repo, err)
			} else {
				data.Message = fmt.Sprintf("Webhook for %s successfully set up/updated!", repo)
				log.Printf("Webhook for %s successfully set up/updated via Web UI.", repo)
			}
		}
	}

	if err := templates.ExecuteTemplate(w, "setup_webhook.html", data); err != nil {
		log.Printf("Error executing template: %v", err)
	}
}

func addAuthHandler(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Message string
		Error   string
	}{}

	if r.Method == http.MethodPost {
		repo := r.FormValue("repo")
		token := r.FormValue("token") // This is the PAT

		if repo == "" || token == "" {
			data.Error = "Repository and Token are required."
		} else {
			log.Printf("Storing authentication for %s via Web UI...", repo)
			if err := storage.StoreRepoAuth(repo, token); err != nil {
				data.Error = fmt.Sprintf("Failed to store authentication data: %v", err)
				log.Printf("Error storing authentication via Web UI for %s: %v", repo, err)
			} else {
				data.Message = fmt.Sprintf("Authentication for %s successfully stored!", repo)
				log.Printf("Authentication for %s successfully stored via Web UI.", repo)
			}
		}
	}

	if err := templates.ExecuteTemplate(w, "add_auth.html", data); err != nil {
		log.Printf("Error executing template: %v", err)
	}
}

func StartWebServer() error {
	http.HandleFunc("/", runHistoryHandler)
	http.HandleFunc("/runs/", runDetailsHandler)
	http.HandleFunc("/setup-webhook", setupWebhookHandler)
	http.HandleFunc("/add-auth", addAuthHandler)

	port := ":8081" // Use a consistent port for the web UI
	fmt.Printf("Web dashboard listening on http://localhost%s...\n", port)
	err := http.ListenAndServe(port, nil)
	if err != nil {
		return fmt.Errorf("failed to start web server: %w", err)
	}
	return nil
}
