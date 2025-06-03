package pipeline

import (
	"fmt"
	"sync"

	"snap-ci/config"
	"snap-ci/executor"
)

// JobResult stores the result of a job execution
type JobResult struct {
	Name   string
	Status string
	Logs   string
}

// ExecutePipeline executes the pipeline defined in the config
func ExecutePipeline(cfg *config.Config) (map[string]JobResult, error) {
	jobResults := make(map[string]JobResult)
	var wg sync.WaitGroup
	jobChan := make(chan config.Job, len(cfg.Jobs))

	//  Populate jobChan with jobs
	for name, job := range cfg.Jobs {
		job.Name = name //  Attach name to job for later use
		jobChan <- job
	}
	close(jobChan)

	//  Process jobs concurrently
	for job := range jobChan {
		wg.Add(1)
		go func(j config.Job) {
			defer wg.Done()
			result := executeJob(j)
			jobResults[j.Name] = result
		}(job)
	}

	wg.Wait()
	return jobResults, nil
}

// executeJob executes a single job
func executeJob(job config.Job) JobResult {
	result := JobResult{
		Name:   job.Name,
		Status: "Success", //  Assume success, change if error
		Logs:   "",
	}

	for _, step := range job.Steps {
		fmt.Printf("  Running step: %s\n", step.Name)
		logs, err := executor.ExecuteStep(step) //  Phase 1: Local execution
		result.Logs += logs + "\n"
		if err != nil {
			result.Status = "Failure"
			result.Logs += "Step failed: " + err.Error()
			break //  Stop on first failure
		}
	}

	return result
}
