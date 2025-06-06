// snap-ci/types/types.go

package types

import "time"

// StepResult stores the result of a single step execution
type StepResult struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Logs   string `json:"logs"`
	// StartTime time.Time `json:"start_time"` // If you add timestamps
	// EndTime   time.Time `json:"end_time"`
}

// JobResult stores the result of a job execution
type JobResult struct {
	Status string                `json:"status"`
	Steps  map[string]StepResult `json:"steps"`
}

// PipelineRun represents a single execution of a CI/CD pipeline.
type PipelineRun struct {
	ID           string               `json:"id"`
	RepoName     string               `json:"repo_name"`
	Branch       string               `json:"branch"`
	CommitSHA    string               `json:"commit_sha"`
	CommitMsg    string               `json:"commit_msg"`
	CommitAuthor string               `json:"commit_author"`
	TriggeredBy  string               `json:"triggered_by"` // User/system that triggered it
	TriggerType  string               `json:"trigger_type"` // New field: e.g., "webhook", "manual", "scheduled"
	Status       string               `json:"status"`       // e.g., "pending", "running", "success", "failure"
	StartTime    time.Time            `json:"start_time"`
	EndTime      time.Time            `json:"end_time"`
	Results      map[string]JobResult `json:"results"`
}
