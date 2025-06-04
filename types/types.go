// snap-ci/types/types.go

package types

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
