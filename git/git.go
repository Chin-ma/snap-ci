// git/git.go

package git

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"encoding/json"
	"io/ioutil"

	"snap-ci/config"
	"snap-ci/pipeline"
	// "snap-ci/storage" // You might want to log webhook events
	// "github.com/go-playground/webhooks/v6/github" // Example library for GitHub webhook parsing and verification
)

type PushEvent struct {
	Ref        string     `json:"ref"`
	Repository Repository `json:"repository"`
}

type Repository struct {
	URL string `json:"url"`
}

// WebhookHandler handles incoming Git webhooks
func WebhookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	payload, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading webhook payload: %v", err)
		http.Error(w, "Error reading payload", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	eventType := r.Header.Get("X-GitHub-Event") // For GitHub
	log.Printf("Received webhook event of type: %s", eventType)

	switch eventType {
	case "push":
		var pushEvent PushEvent
		if err := json.Unmarshal(payload, &pushEvent); err != nil {
			log.Printf("Error unmarshalling push event: %v", err)
			http.Error(w, "Error unmarshalling push event", http.StatusBadRequest)
			return
		}

		repoURL := pushEvent.Repository.URL
		fullRef := pushEvent.Ref
		fmt.Printf("Received push event for: %s on branch: %s\n", repoURL, fullRef)

		// TODO: Clone the repository and execute the pipeline
		if repoURL != "" && fullRef != "" {
			if err := cloneRepo(repoURL, fullRef); err != nil {
				log.Printf("Error cloning repository: %v", err)
				http.Error(w, "Failed to clone repository", http.StatusInternalServerError)
				return
			}
			cfg, err := config.LoadConfig("temp_repo/.ci.yaml")
			if err != nil {
				log.Printf("Error loading .ci.yaml: %v", err)
				http.Error(w, "Failed to load .ci.yaml", http.StatusInternalServerError)
				return
			}
			_, err = pipeline.ExecutePipeline(cfg)
			if err != nil {
				log.Printf("Pipeline execution failed: %v", err)
				http.Error(w, "Pipeline execution failed", http.StatusInternalServerError)
				return
			}
		}
	case "ping":
		fmt.Println("Received ping event. Responding with OK.")
		w.WriteHeader(http.StatusOK)
		return
	default:
		fmt.Printf("Received unhandled webhook event: %s\n", eventType)
		w.WriteHeader(http.StatusOK)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Println("Webhook received and processed")
}

// cloneRepo clones the Git repository
func cloneRepo(repoURL string, fullRef string) error { // Changed parameter name to fullRef
	// Remove the temp_repo directory if it exists
	if _, err := os.Stat("temp_repo"); !os.IsNotExist(err) {
		log.Println("Removing existing temp_repo directory")
		if err := os.RemoveAll("temp_repo"); err != nil {
			return fmt.Errorf("failed to remove existing temp_repo: %w", err)
		}
	}
	var branch string
	parts := strings.Split(fullRef, "/")
	if len(parts) > 2 && parts[1] == "heads" {
		branch = parts[2] // Extract the branch name
	} else {
		branch = "main" // Default to main if extraction fails or ref is unexpected
		log.Printf("Warning: Could not extract branch name from ref '%s', defaulting to 'main'", fullRef)
	}

	cmd := exec.Command("git", "clone", "-b", branch, repoURL, "temp_repo")
	log.Printf("Executing: %s", cmd.String())

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("git clone error: %v, output: %s", err, string(output))
		return fmt.Errorf("git clone failed: %w", err)
	}
	log.Printf("git clone output: %s", string(output))
	return nil
}

// StartWebhookListener starts the HTTP server to listen for webhooks
func StartWebhookListener() error {
	http.HandleFunc("/webhook", WebhookHandler)
	port := ":8080"
	fmt.Printf("Listening for webhooks on port %s...\n", port)
	err := http.ListenAndServe(port, nil)
	if err != nil {
		return fmt.Errorf("failed to start webhook listener: %w", err)
	}
	return nil
}
