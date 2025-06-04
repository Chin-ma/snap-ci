// main.go

package main

import (
	"fmt"
	"log"
	"os"

	"snap-ci/config"
	"snap-ci/git"
	"snap-ci/pipeline"
	"snap-ci/storage"
	"snap-ci/web"

	"github.com/urfave/cli/v2" // Or Cobra
)

func main() {
	app := &cli.App{
		Name:    "snapci",
		Usage:   "A lightweight CI/CD pipeline tool",
		Version: "0.1.0",
		Commands: []*cli.Command{
			{
				Name:  "run",
				Usage: "Trigger a pipeline run (for testing)",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "config", Value: ".ci.yaml", Usage: "Path to .ci.yaml"},
				},
				Action: func(c *cli.Context) error {
					cfgPath := c.String("config")
					cfg, err := config.LoadConfig(cfgPath)
					if err != nil {
						return err
					}

					//  Normally, this would be triggered by a webhook
					//  For testing, we trigger it manually
					jobResults, err := pipeline.ExecutePipeline(*cfg)
					if err != nil {
						return err
					}

					//  Store results and display in CLI
					// FIX: Provide placeholder values for the new arguments required by storage.StoreRun
					if err := storage.StoreRun(
						cfg,
						jobResults,
						"manual-run/repo",         // Placeholder
						"manual-branch",           // Placeholder
						"manual-sha",              // Placeholder
						"Manual pipeline trigger", // Placeholder
						"manual-user",             // Placeholder
						"cli-user",                // Placeholder
					); err != nil {
						return err
					}
					storage.DisplayRunResults(jobResults)

					return nil
				},
			},
			{
				Name:  "webhooks",
				Usage: "Start the webhook listener",
				Action: func(c *cli.Context) error {
					//  Start the webhook listener
					git.StartWebhookListener()
					return nil
				},
			},
			{
				Name:  "logs",
				Usage: "View logs for a run",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Usage: "Id of the run view logs for"},
				},
				Action: func(c *cli.Context) error {
					runID := c.String("id")
					if runID == "" {
						return cli.Exit("Please provide a run ID using the --id flag", 1)
					}
					if err := displayRunLogs(runID); err != nil {
						return err
					}
					return nil //  Implement log viewing logic here
				},
			},
			{
				Name:  "status",
				Usage: "View the status of recent or specific runs",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Usage: "Id of the run to view status for"},
					&cli.BoolFlag{Name: "recent", Usage: "View the status of recent runs"},
				},
				Action: func(c *cli.Context) error {
					//  Implement status viewing logic here
					fmt.Println("Status command not yet implemented")
					return nil
				},
			},
			{
				Name:  "web",
				Usage: "Start the web UI",
				Action: func(c *cli.Context) error {
					web.StartWebServer()
					return nil
				},
			},
		},
		Action: func(c *cli.Context) error {
			return c.App.Command("run").Run(c)
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func displayRunLogs(runID string) error {
	run, err := storage.GetRun(runID)
	if err != nil {
		return err
	}
	fmt.Printf("Logs for Run ID: %s\n", runID)
	// Display new run metadata
	fmt.Printf("  Repository: %s\n", run.RepoName)
	fmt.Printf("  Branch: %s\n", run.Branch)
	fmt.Printf("  Commit: %s - %s\n", run.CommitSHA, run.CommitMsg)
	fmt.Printf("  Author: %s\n", run.CommitAuthor)
	fmt.Printf("  Triggered By: %s\n", run.TriggeredBy)
	fmt.Println("---")

	for jobName, result := range run.Results {
		fmt.Printf("Job: %s - Status: %s\n", jobName, result.Status)
		for stepName, stepResult := range result.Steps {
			fmt.Printf("Step: %s - Status: %s\n", stepName, stepResult.Status)
			fmt.Printf("Logs:\n%s\n", stepResult.Logs)
		}
		fmt.Println("---")
	}
	return nil
}
