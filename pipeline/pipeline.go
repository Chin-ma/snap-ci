package pipeline

import (
	"log"
	"snap-ci/config"
	"snap-ci/executor"
	"snap-ci/types"
)

// ExecutePipeline executes the pipeline defined in the config
func ExecutePipeline(cfg config.Config) (map[string]types.JobResult, error) {
	jobResults := make(map[string]types.JobResult)

	// startTime := time.Now() // If you add timestamps
	for jobName, job := range cfg.Jobs {
		// jobStartTime := time.Now() // If you add timestamps
		jobResult := types.JobResult{
			Status: "Success",
			Steps:  make(map[string]types.StepResult),
		}

		for _, step := range job.Steps {
			// stepStartTime := time.Now() // If you add timestamps
			stepResult, err := executor.ExecuteStep(executor.Step(step), "temp_repo") // Assuming "temp_repo" is the working dir
			// stepEndTime := time.Now()

			jobResult.Steps[step.Name] = stepResult // Store the StepResult

			if err != nil {
				jobResult.Status = "Failure"
				log.Printf("Job '%s', Step '%s' failed: %v", jobName, step.Name, err)
				break // Stop executing steps in this job
			}
			// Optionally log step success
			log.Printf("Job '%s', Step '%s' succeeded", jobName, step.Name)

		}
		// jobEndTime := time.Now()
		jobResults[jobName] = jobResult
	}
	// endTime := time.Now()

	return jobResults, nil
}
