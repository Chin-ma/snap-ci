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
				Name:  "webhook",
				Usage: "Manage GitHub webhooks for automatic registration",
				Subcommands: []*cli.Command{
					{
						Name:  "setup",
						Usage: "Set up or update a GitHub webhook with the current ngrok URL",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "repo",
								Usage:    "GitHub repository in the format 'owner/repo-name' (e.g., 'myorg/myproject')",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "token",
								Usage:    "GitHub Personal Access Token with 'repo:hooks' scope",
								Required: true,
								EnvVars:  []string{"GITHUB_TOKEN"}, // Allow token from env var
							},
						},
						Action: func(c *cli.Context) error {
							repo := c.String("repo")
							token := c.String("token")

							log.Printf("Attempting to set up webhook for %s...", repo)
							// Call a function in your 'web' package to handle the actual setup
							// This function would fetch the ngrok URL and interact with GitHub API
							if err := git.SetupGitHubWebhook(repo, token); err != nil {
								return fmt.Errorf("failed to set up GitHub webhook: %w", err)
							}
							log.Printf("Webhook for %s successfully set up/updated.", repo)
							return nil
						},
					},
					// Add other webhook management subcommands if needed (e.g., "delete", "list")
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
			{
				Name:  "auth",
				Usage: "Manage authentication credentials for private repositories",
				Subcommands: []*cli.Command{
					{
						Name:  "add",
						Usage: "Add Github Personal Access Token (PAT) for a private repository",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "repo",
								Usage:    "GitHub repository in the format 'owner/repo-name' (e.g., 'myorg/myproject')",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "token",
								Usage:    "GitHub Personal Access Token with 'repo:hooks' scope",
								Required: true,
								EnvVars:  []string{"GITHUB_TOKEN"}, // Allow token from env var
							},
						},
						Action: func(c *cli.Context) error {
							repo := c.String("repo")
							token := c.String("token")
							if token == "" {
								return cli.Exit("Github PAT is required. Use --token flag or set GITHUB_TOKEN env var", 1)
							}
							log.Printf("Storing Authentication data for %s...", repo)
							if err := storage.StoreRepoAuth(repo, token); err != nil {
								return fmt.Errorf("failed to store authentication data: %w", err)
							}
							log.Printf("Authentication for %s successfully stored. Use this repo in git clone operations", repo)
							return nil
						},
					},
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
