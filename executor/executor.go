// executor/executor.go

package executor

import (
	"bytes"
	"fmt" // Import fmt for better error formatting
	"log"
	"os/exec"
	"snap-ci/types"
	"strings" // Import strings for trimming whitespace
	// "time" // If you add timestamps
)

// Step represents a single execution step.
type Step struct { // Define the Step struct here or import it if defined elsewhere
	Name string `yaml:"name"`
	Run  string `yaml:"run"`
}

// ExecuteStep executes a single step in the pipeline.
func ExecuteStep(step Step, workingDir string) (types.StepResult, error) {
	// startTime := time.Now() // If you add timestamps

	cmd := exec.Command("bash", "-c", step.Run)
	cmd.Dir = workingDir

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	// endTime := time.Now() // If you add timestamps

	// Capture both stdout and stderr
	stdout := stdoutBuf.String()
	stderr := stderrBuf.String()
	logs := stdout + stderr

	status := "Success"
	if err != nil {
		status = "Failure"
		log.Printf("Step '%s' failed: %v", step.Name, err)
		// Include stderr in the error message for more context
		return types.StepResult{}, fmt.Errorf("step '%s' failed: %v, stderr: %s", step.Name, err, strings.TrimSpace(stderr))
	}

	stepResult := types.StepResult{
		Name:   step.Name,
		Status: status,
		Logs:   logs,
		// StartTime: startTime, // If you add timestamps
		// EndTime:   endTime,
	}

	// Log the output (optional, but helpful for debugging)
	log.Printf("Step '%s' output:\n%s", step.Name, logs)

	return stepResult, nil
}
