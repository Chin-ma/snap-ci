package executor

import (
	"fmt"
	"os/exec"

	"snap-ci/config"
)

// ExecuteStep executes a step directly on the host
func ExecuteStep(step config.Step) (string, error) {
	cmd := exec.Command("/bin/sh", "-c", step.Run)
	cmd.Dir = "./temp_repo" // Execute commands within the cloned repository

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("command failed: %w\nOutput: %s", err, string(output))
	}

	return string(output), nil
}
