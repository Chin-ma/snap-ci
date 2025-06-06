package git

import (
	"fmt"
	"log"
	"path/filepath" // Still useful for joining paths like .ci.yaml
	"strings"
	"time"

	"snap-ci/config"
	"snap-ci/pipeline"
	"snap-ci/storage" // This package contains storage.GetRepoAuth, storage.StoreRun etc.
	"snap-ci/types"   // This package contains types.JobResult, types.StepResult
)

func TriggerManualRun(repoName, branch, commitSHA string) error {
	runID := fmt.Sprintf("manual-%s-%d", strings.ReplaceAll(repoName, "/", "-"), time.Now().UnixNano())

	// 1. Determine Repository URL and Authentication
	repoURL := fmt.Sprintf("https://github.com/%s.git", repoName) // Default to public HTTPS
	// storage.GetRepoAuth is exported, so it can be called directly.
	if patAuth, err := storage.GetRepoAuth(repoName); err == nil && patAuth != nil && patAuth.GithubToken != "" {
		repoURL = fmt.Sprintf("https://oauth2:%s@github.com/%s.git", patAuth.GithubToken, repoName)
		log.Printf("Using stored GitHub PAT for cloning %s.", repoName)
	} else if err != nil {
		log.Printf("No stored authentication found for %s (%v). Cloning might fail for private repos.", repoName, err)
	} else {
		log.Printf("No stored authentication found for %s. Cloning might fail for private repos.", repoName)
	}

	// 2. Determine the ref to clone (branch or default)
	cloneRef := branch
	if cloneRef == "" {
		cloneRef = "main" // Default to main if no branch provided
	}
	fullRef := fmt.Sprintf("refs/heads/%s", cloneRef) // git.cloneRepo expects "refs/heads/branch-name"

	// 3. Clone the Repository
	log.Printf("Cloning %s (ref: %s) into 'temp_repo'...", repoName, cloneRef)
	if err := cloneRepo(repoURL, fullRef); err != nil {
		return fmt.Errorf("failed to clone repository %s (ref: %s): %w", repoName, cloneRef, err)
	}

	// The `repoDir` for subsequent operations is implicitly "temp_repo"
	const currentRepoWorkingDir = "temp_repo"

	// 4. If a specific commit SHA is provided, check it out after cloning the branch
	if commitSHA != "" {
		log.Printf("Checking out specific commit '%s' in %s...", commitSHA, currentRepoWorkingDir)
		// git.CheckoutCommit is exported.
		if err := CheckoutCommit(currentRepoWorkingDir, commitSHA); err != nil {
			return fmt.Errorf("failed to checkout commit '%s' in %s: %w", commitSHA, currentRepoWorkingDir, err)
		}
	} else {
		// Ensure the branch derived from `branch` input is checked out if no commit SHA
		// This handles cases where `git.cloneRepo` might default to main, but a specific `branch` was requested.
		// git.CheckoutBranch is exported.
		if branch != "" && branch != "main" && branch != "master" { // Avoid redundant checkout for common default branches
			log.Printf("Ensuring branch '%s' is checked out in %s...", branch, currentRepoWorkingDir)
			if err := CheckoutBranch(currentRepoWorkingDir, branch); err != nil {
				return fmt.Errorf("failed to checkout branch '%s' in %s: %w", branch, currentRepoWorkingDir, err)
			}
		}
	}

	// Get the actual commit SHA and branch name after all checkout operations
	// git.GetCurrentCommit and git.GetCurrentBranch are exported.
	effectiveCommitSHA := commitSHA // Start with provided SHA, or update from HEAD
	if currentCommit, err := GetCurrentCommit(currentRepoWorkingDir); err != nil {
		log.Printf("Warning: Could not get current commit SHA from %s: %v", currentRepoWorkingDir, err)
		effectiveCommitSHA = "unknown"
	} else {
		effectiveCommitSHA = currentCommit
	}

	effectiveBranch := branch // Start with provided branch, or update from HEAD
	if currentBranch, err := GetCurrentBranch(currentRepoWorkingDir); err == nil {
		effectiveBranch = currentBranch
	} else {
		log.Printf("Warning: Could not get current branch name from %s: %v", currentRepoWorkingDir, err)
		if effectiveBranch == "" {
			effectiveBranch = "unknown"
		}
	}

	// 5. Load the .ci.yaml configuration
	configPath := filepath.Join(currentRepoWorkingDir, ".ci.yaml")
	// config.LoadConfig is expected to be exported.
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load pipeline configuration from %s: %w", configPath, err)
	}

	// 6. Get Commit Details for Run Metadata
	var commitAuthor, commitMsg string
	if effectiveCommitSHA != "unknown" && effectiveCommitSHA != "" {
		// git.GetCommitDetails is exported.
		commitAuthor, commitMsg, err = GetCommitDetails(currentRepoWorkingDir, effectiveCommitSHA)
		if err != nil {
			log.Printf("Warning: Could not get commit details for SHA '%s': %v. Using defaults.", effectiveCommitSHA, err)
			commitAuthor = "N/A"
			commitMsg = "Manual trigger"
		}
	} else {
		commitAuthor = "N/A"
		commitMsg = "Manual trigger (no specific commit SHA determined)"
	}

	// 7. Initialize PipelineRun Object
	pipelineRun := &types.PipelineRun{
		ID:           runID,
		RepoName:     repoName,
		Branch:       effectiveBranch,
		CommitSHA:    effectiveCommitSHA,
		CommitMsg:    commitMsg,
		CommitAuthor: commitAuthor,
		TriggeredBy:  "CLI User",
		TriggerType:  "manual",
		Status:       "pending",
		StartTime:    time.Now(),
		Results:      make(map[string]types.JobResult),
	}

	log.Printf("Executing manually triggered pipeline run %s for commit '%s' on branch '%s'...",
		pipelineRun.ID, pipelineRun.CommitSHA, pipelineRun.Branch)

	// 8. Execute the Pipeline (calling pipeline.ExecutePipeline as it currently is)
	jobResultsFromPipeline, err := pipeline.ExecutePipeline(*cfg)
	if err != nil {
		log.Printf("Manually triggered pipeline run %s failed during pipeline execution: %v", pipelineRun.ID, err)
		pipelineRun.Status = "failure"
	} else {
		// Copy results from types.JobResult (returned by pipeline.ExecutePipeline) to run.JobResult
		// for the PipelineRun object.
		for jobName, result := range jobResultsFromPipeline {
			pipelineRun.Results[jobName] = types.JobResult{
				Status: result.Status,
				Steps:  make(map[string]types.StepResult),
			}
			for stepName, stepResult := range result.Steps {
				pipelineRun.Results[jobName].Steps[stepName] = types.StepResult{
					Status: stepResult.Status,
					Logs:   stepResult.Logs,
				}
			}
		}

		// Determine overall pipeline status based on returned jobResults
		allJobsSuccess := true
		for _, jobResult := range jobResultsFromPipeline {
			if jobResult.Status == "Failure" { // Use case-sensitive "Failure" as per pipeline.go and types.JobResult
				allJobsSuccess = false
				break
			}
		}

		if allJobsSuccess {
			pipelineRun.Status = "success"
		} else {
			pipelineRun.Status = "failure"
		}
	}
	pipelineRun.EndTime = time.Now()

	// 9. Store the PipelineRun Results
	// storage.StoreRun is exported, so it can be called directly.
	if err := storage.StoreRun(
		cfg,
		pipelineRun.Results, // Pass the results stored in pipelineRun
		pipelineRun.RepoName,
		pipelineRun.Branch,
		pipelineRun.CommitSHA,
		pipelineRun.CommitMsg,
		pipelineRun.CommitAuthor,
		pipelineRun.TriggeredBy,
	); err != nil {
		log.Printf("Warning: Failed to store manual run results for %s: %v", pipelineRun.ID, err)
	}

	log.Printf("Manually triggered pipeline run %s finished with status: %s", pipelineRun.ID, pipelineRun.Status)
	return nil
}
