package git

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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

// NgrokTunnel represents a single tunnel returned by the Ngrok API
type NgrokTunnel struct {
	PublicURL string `json:"public_url"`
	Proto     string `json:"proto"`
}

// NgrokTunnelsResponse represents the full response from the Ngrok API's /api/tunnels endpoint
type NgrokTunnelsResponse struct {
	Tunnels []NgrokTunnel `json:"tunnels"`
}

// GetNgrokPublicURL queries the local Ngrok API to get the public HTTPS tunnel URL.
func GetNgrokPublicURL() (string, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("http://127.0.0.1:4040/api/tunnels")
	if err != nil {
		return "", fmt.Errorf("failed to query Ngrok API (is Ngrok running?): %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Ngrok API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var data NgrokTunnelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", fmt.Errorf("failed to decode Ngrok API response: %w", err)
	}

	for _, tunnel := range data.Tunnels {
		if tunnel.Proto == "https" {
			return tunnel.PublicURL, nil
		}
	}
	return "", fmt.Errorf("no public HTTPS tunnel found in Ngrok API response. Ensure Ngrok is forwarding an HTTPS tunnel (e.g., ngrok http 8080)")
}

// RegisterGithubWebhook registers or updates a webhook on GitHub.
// It checks if a webhook exists and attempts to update it, otherwise creates a new one.
func RegisterGithubWebhook(owner, repo, webhookURL, githubToken string) error {
	// First, check if a webhook already exists for this URL
	existingWebhooksURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/hooks", owner, repo)
	req, err := http.NewRequest(http.MethodGet, existingWebhooksURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create get webhooks request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", fmt.Sprintf("token %s", githubToken))
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to get existing webhooks: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GitHub API (get webhooks) returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var hooks []struct {
		ID     int64 `json:"id"`
		Config struct {
			URL string `json:"url"`
		} `json:"config"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&hooks); err != nil {
		return fmt.Errorf("failed to decode existing webhooks response: %w", err)
	}

	var existingHookID int64 = 0
	for _, hook := range hooks {
		// GitHub might append a trailing slash, so normalize for comparison
		if strings.TrimSuffix(hook.Config.URL, "/") == strings.TrimSuffix(webhookURL, "/") {
			existingHookID = hook.ID
			break
		}
	}

	hookConfig := map[string]interface{}{
		"name":   "web",
		"active": true,
		"events": []string{"push"},
		"config": map[string]string{
			"url":          webhookURL,
			"content_type": "json",
			"insecure_ssl": "0", // Always set to "0" for security unless absolutely necessary
		},
	}

	var apiMethod string
	var apiTargetURL string
	if existingHookID != 0 {
		apiMethod = http.MethodPatch // Update existing webhook
		apiTargetURL = fmt.Sprintf("https://api.github.com/repos/%s/%s/hooks/%d", owner, repo, existingHookID)
		log.Printf("Updating existing webhook (ID: %d) for %s/%s to %s", existingHookID, owner, repo, webhookURL)
	} else {
		apiMethod = http.MethodPost // Create new webhook
		apiTargetURL = fmt.Sprintf("https://api.github.com/repos/%s/%s/hooks", owner, repo)
		log.Printf("Creating new webhook for %s/%s at %s", owner, repo, webhookURL)
	}

	body, err := json.Marshal(hookConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook config: %w", err)
	}

	req, err = http.NewRequest(apiMethod, apiTargetURL, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create GitHub API request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", fmt.Sprintf("token %s", githubToken))
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err = client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send GitHub API request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 { // 200 OK for PATCH, 201 Created for POST
		log.Printf("Successfully set up webhook for %s/%s at %s", owner, repo, webhookURL)
		return nil
	} else {
		return fmt.Errorf("GitHub API returned error status %d: %s", resp.StatusCode, string(respBody))
	}
}

// SetupGitHubWebhook orchestrates fetching the ngrok URL and registering it with GitHub.
func SetupGitHubWebhook(repoFullName, githubToken string) error {
	log.Println("Fetching Ngrok public URL...")
	ngrokURL, err := GetNgrokPublicURL()
	if err != nil {
		return fmt.Errorf("could not get Ngrok public URL: %w", err)
	}
	log.Printf("Ngrok public URL: %s", ngrokURL)

	parts := strings.Split(repoFullName, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid repository format: %s. Expected 'owner/repo-name'", repoFullName)
	}
	owner := parts[0]
	repoName := parts[1]

	// Append the webhook path to the ngrok URL
	fullWebhookURL := ngrokURL + "/webhook"

	log.Printf("Attempting to register GitHub webhook for %s/%s with URL: %s", owner, repoName, fullWebhookURL)
	if err := RegisterGithubWebhook(owner, repoName, fullWebhookURL, githubToken); err != nil {
		return fmt.Errorf("failed to register GitHub webhook: %w", err)
	}

	log.Printf("GitHub webhook successfully configured for %s/%s.", repoFullName)
	return nil
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

	// --- NEW: Handle private repository authentication ---
	// Extract owner/repo name from cloneURL for auth lookup
	// e.g., "https://github.com/owner/repo.git" -> "owner/repo"
	repoFullName := ""
	if strings.HasPrefix(repoURL, "https://github.com/") {
		trimmed := strings.TrimPrefix(repoURL, "https://github.com/")
		trimmed = strings.TrimSuffix(trimmed, ".git")
		repoFullName = trimmed
	} else {
		// Handle other git providers/protocols if necessary
		log.Printf("Warning: Unsupported repository URL format for automatic authentication: %s. Proceeding without stored PAT.", repoURL)
	}

	auth := &storage.RepoAuth{} // Initialize auth to nil or default
	if repoFullName != "" {
		var err error
		auth, err = storage.GetRepoAuth(repoFullName)
		if err != nil {
			log.Printf("No stored authentication found for %s: %v. Attempting to clone without token (might fail for private repos).", repoFullName, err)
			// If no auth found, proceed without it; git will prompt or fail
		}
	}

	cloneCmdArgs := []string{"clone", "-b", branch}
	finalRepoURL := repoURL

	if auth != nil && auth.GithubToken != "" {
		// For HTTPS, embed the token directly into the URL
		// Format: https://oauth2:<token>@github.com/owner/repo.git
		if strings.HasPrefix(repoURL, "https://github.com/") {
			finalRepoURL = fmt.Sprintf("https://oauth2:%s@%s", auth.GithubToken, strings.TrimPrefix(repoURL, "https://"))
			log.Printf("Using stored GitHub PAT for cloning %s", repoFullName)
		} else {
			log.Printf("Warning: Stored PAT is for GitHub, but repoURL is not GitHub HTTPS: %s. Proceeding without embedding token.", repoURL)
		}
	}

	cloneCmdArgs = append(cloneCmdArgs, finalRepoURL, "temp_repo")

	cmd := exec.Command("git", cloneCmdArgs...) // Use the slice of arguments
	log.Printf("Executing: git %s", strings.Join(cloneCmdArgs, " "))

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

func CheckoutBranch(repoDir, branch string) error {
	cmd := exec.Command("git", "checkout", branch)
	cmd.Dir = repoDir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to checkout branch %s: %w, stderr: %s", branch, err, stderr.String())
	}
	return nil
}

func CheckoutCommit(repoDir, commitSHA string) error {
	cmd := exec.Command("git", "checkout", commitSHA)
	cmd.Dir = repoDir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to checkout commit %s: %w, stderr: %s", commitSHA, err, stderr.String())
	}
	return nil
}

func GetCurrentCommit(repoDir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = repoDir
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current commit: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func GetCurrentBranch(repoDir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = repoDir
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func GetCommitDetails(repoDir, commitSHA string) (author string, message string, err error) {
	if commitSHA == "" {
		return "", "", fmt.Errorf("commit SHA cannot be empty")
	}

	cmdAuthor := exec.Command("git", "log", "-1", "--format=%an", commitSHA)
	cmdAuthor.Dir = repoDir
	authorOutput, err := cmdAuthor.Output()
	if err != nil {
		return "", "", fmt.Errorf("failed to get commit author: %w", err)
	}
	author = strings.TrimSpace(string(authorOutput))

	cmdMessage := exec.Command("git", "log", "-1", "--format=%B", commitSHA)
	cmdMessage.Dir = repoDir
	messageOutput, err := cmdMessage.Output()
	if err != nil {
		return "", "", fmt.Errorf("failed to get commit message: %w", err)
	}
	message = strings.TrimSpace(string(messageOutput))
	return author, message, nil
}

func GetCommitSHAFromBranch(repoDir, branch string) (string, error) {
	cmd := exec.Command("git", "rev-parse", branch)
	cmd.Dir = repoDir
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get commit SHA from branch: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}
