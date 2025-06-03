package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"snap-ci/config"
	"snap-ci/pipeline"
)

const (
	runMetadataDir = "run_metadata"
)

// RunMetadata stores metadata about a pipeline run
type RunMetadata struct {
	ID        string                        `json:"id"`
	Config    config.Config                 `json:"config"`
	Results   map[string]pipeline.JobResult `json:"results"`
	StartTime time.Time                     `json:"start_time"`
	EndTime   time.Time                     `json:"end_time"`
	Status    string                        `json:"status"`
}

// StoreRun stores the results of a pipeline run to a JSON file
func StoreRun(cfg *config.Config, results map[string]pipeline.JobResult) error {
	err := os.MkdirAll(runMetadataDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create run metadata directory: %w", err)
	}

	runID := time.Now().Format("20060102150405") // Unique ID based on timestamp
	metadata := RunMetadata{
		ID:        runID,
		Config:    *cfg,
		Results:   results,
		StartTime: time.Now(),
		EndTime:   time.Now(),
		Status:    calculateOverallStatus(results),
	}

	filename := filepath.Join(runMetadataDir, fmt.Sprintf("run_%s.json", runID))
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create metadata file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	err = encoder.Encode(metadata)
	if err != nil {
		return fmt.Errorf("failed to encode metadata to JSON: %w", err)
	}

	fmt.Printf("Run metadata stored in: %s\n", filename)
	return nil
}

func calculateOverallStatus(results map[string]pipeline.JobResult) string {
	overallStatus := "Success"
	for _, result := range results {
		if result.Status == "Failure" {
			overallStatus = "Failure"
			break
		}
	}
	return overallStatus
}

// GetRun retrieves the metadata for a specific run ID
func GetRun(runID string) (*RunMetadata, error) {
	filename := filepath.Join(runMetadataDir, fmt.Sprintf("run_%s.json", runID))
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("run with ID '%s' not found", runID)
		}
		return nil, fmt.Errorf("failed to open metadata file: %w", err)
	}
	defer file.Close()

	var metadata RunMetadata
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to decode metadata from JSON: %w", err)
	}

	return &metadata, nil
}

// GetRecentRuns retrieves a list of the most recent pipeline runs
func GetRecentRuns(limit int) ([]RunMetadata, error) {
	files, err := os.ReadDir(runMetadataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []RunMetadata{}, nil // No runs yet
		}
		return nil, fmt.Errorf("failed to read run metadata directory: %w", err)
	}

	var runs []RunMetadata
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".json" && len(file.Name()) > 8 && file.Name()[:4] == "run_" {
			runID := file.Name()[4 : len(file.Name())-5]
			metadata, err := GetRun(runID)
			if err == nil {
				runs = append(runs, *metadata)
			} else {
				fmt.Printf("Error reading run %s: %v\n", runID, err)
			}
		}
	}

	// Sort runs by start time in descending order (most recent first)
	sort.Slice(runs, func(i, j int) bool {
		return runs[i].StartTime.After(runs[j].StartTime)
	})

	if len(runs) > limit {
		return runs[:limit], nil
	}
	return runs, nil
}

// DisplayRunResults displays the results in the CLI (remains the same)
func DisplayRunResults(results map[string]pipeline.JobResult) {
	fmt.Println("Pipeline Results:")
	for jobName, result := range results {
		fmt.Printf("  %s: %s\n", jobName, result.Status)
		fmt.Printf("  Logs:\n%s\n", result.Logs)
	}
}
