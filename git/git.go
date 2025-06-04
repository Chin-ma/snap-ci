// git/git.go

package git

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"snap-ci/config"
	"snap-ci/pipeline"
	"snap-ci/storage"
	// Import the new types package
)

// Define a more comprehensive PushEvent struct to match GitHub's payload
// This structure is derived from common GitHub Push event payloads
type PushEvent struct {
	Ref        string     `json:"ref"`
	Before     string     `json:"before"`
	After      string     `json:"after"`
	Repository Repository `json:"repository"`
	Pusher     Pusher     `json:"pusher"`
	Sender     Sender     `json:"sender"`
	Created    bool       `json:"created"`
	Deleted    bool       `json:"deleted"`
	Forced     bool       `json:"forced"`
	BaseRef    *string    `json:"base_ref"` // Can be null
	Compare    string     `json:"compare"`
	Commits    []Commit   `json:"commits"`
	HeadCommit *Commit    `json:"head_commit"` // Can be null
}

type Repository struct {
	ID            int64   `json:"id"`
	NodeID        string  `json:"node_id"`
	Name          string  `json:"name"`
	FullName      string  `json:"full_name"`
	Private       bool    `json:"private"`
	Owner         Owner   `json:"owner"`
	HTMLURL       string  `json:"html_url"`
	Description   *string `json:"description"` // Can be null
	Fork          bool    `json:"fork"`
	URL           string  `json:"url"`       // API URL
	CloneURL      string  `json:"clone_url"` // The URL to clone the repository
	DefaultBranch string  `json:"default_branch"`
}

type Owner struct {
	Name  string `json:"name"`  // For push event, often 'pusher's name'
	Email string `json:"email"` // For push event, often 'pusher's email'
	Login string `json:"login"` // GitHub username
	ID    int64  `json:"id"`
	URL   string `json:"url"`
	Type  string `json:"type"` // e.g., "User"
}

type Pusher struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Sender struct {
	Login     string `json:"login"`
	ID        int64  `json:"id"`
	NodeID    string `json:"node_id"`
	AvatarURL string `json:"avatar_url"`
	HTMLURL   string `json:"html_url"`
	Type      string `json:"type"`
}

type Commit struct {
	ID        string       `json:"id"`
	TreeID    string       `json:"tree_id"`
	Distinct  bool         `json:"distinct"`
	Message   string       `json:"message"`
	Timestamp time.Time    `json:"timestamp"`
	URL       string       `json:"url"`
	Author    CommitAuthor `json:"author"`
	Committer CommitAuthor `json:"committer"`
	Added     []string     `json:"added"`
	Removed   []string     `json:"removed"`
	Modified  []string     `json:"modified"`
}

type CommitAuthor struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Username string `json:"username"`
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
			log.Printf("Payload content: %s", string(payload)) // Log the full payload for debugging
			http.Error(w, "Error unmarshalling push event", http.StatusBadRequest)
			return
		}

		// Use clone_url for cloning as it's the most reliable URL for git operations
		repoURL := pushEvent.Repository.CloneURL
		fullRef := pushEvent.Ref
		log.Printf("Received push event for: %s on branch: %s", repoURL, fullRef)

		// Check if it's a delete event (e.g., branch deleted)
		if pushEvent.Deleted {
			log.Printf("Ignoring deleted ref: %s", fullRef)
			w.WriteHeader(http.StatusOK)
			fmt.Println("Webhook received and processed (ref deleted)")
			return
		}

		if repoURL != "" && fullRef != "" {
			if err := cloneRepo(repoURL, fullRef); err != nil { // cloneRepo handles branch extraction
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

			jobResults, err := pipeline.ExecutePipeline(*cfg) // ExecutePipeline returns map[string]types.JobResult
			if err != nil {
				log.Printf("Pipeline execution failed: %v", err)
				http.Error(w, "Pipeline execution failed", http.StatusInternalServerError)
				return
			}

			// Extract new metadata from pushEvent
			repoName := pushEvent.Repository.FullName
			branch := strings.TrimPrefix(fullRef, "refs/heads/") // "refs/heads/main" -> "main"
			commitSHA := ""
			commitMsg := ""
			commitAuthor := ""
			if pushEvent.HeadCommit != nil {
				commitSHA = pushEvent.HeadCommit.ID
				commitMsg = pushEvent.HeadCommit.Message
				commitAuthor = pushEvent.HeadCommit.Author.Name
			}
			triggeredBy := pushEvent.Sender.Login

			// Call StoreRun with the new metadata fields
			if err := storage.StoreRun(
				cfg,
				jobResults,
				repoName,
				branch,
				commitSHA,
				commitMsg,
				commitAuthor,
				triggeredBy,
			); err != nil {
				log.Printf("Error storing run results: %v", err)
				http.Error(w, "Failed to store run results", http.StatusInternalServerError)
				return
			}

			storage.DisplayRunResults(jobResults) // Display in CLI output
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
func cloneRepo(repoURL string, fullRef string) error {
	if _, err := os.Stat("temp_repo"); !os.IsNotExist(err) {
		log.Println("Removing existing temp_repo directory")
		if err := os.RemoveAll("temp_repo"); err != nil {
			return fmt.Errorf("failed to remove existing temp_repo: %w", err)
		}
	}

	var branch string
	parts := strings.Split(fullRef, "/")
	if len(parts) > 2 && parts[1] == "heads" {
		branch = parts[2]
	} else if len(parts) > 0 && parts[0] == "refs" && len(parts) > 2 && parts[1] == "tags" {
		log.Printf("Ignoring tag ref for cloning: %s", fullRef)
		return fmt.Errorf("tag ref received, not cloning: %s", fullRef)
	} else {
		log.Printf("Warning: Could not extract branch name from ref '%s', defaulting to 'main'", fullRef)
		branch = "main"
	}

	cmd := exec.Command("git", "clone", "-b", branch, repoURL, "temp_repo")
	log.Printf("Executing: %s", cmd.String())

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("git clone error: %v, output: %s", err, string(output))
		return fmt.Errorf("git clone failed: %w, output: %s", err, string(output))
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
