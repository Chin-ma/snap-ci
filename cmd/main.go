// main.go

package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"snap-ci/config"
	"snap-ci/git"
	"snap-ci/pipeline"
	"snap-ci/storage"
	"snap-ci/web"

	"github.com/urfave/cli/v2" // Or Cobra
)

const (
	webhookListenerPort = 8080
	ngrokAPIPort        = 4040
)

func ensureNgrokInstalled() error {
	_, err := exec.LookPath("ngrok")
	if err != nil {
		log.Println("ngrok not found in system path")
		log.Println("Please install ngrok from https://ngrok.com/download and ensure it's added to your system path.")
		log.Println("Also, remember to authenticate ngrok once: `ngrok config add-authtoken <your_ngrok_auth_token>`")
		return fmt.Errorf("ngrok not installed or not found in system path: %w", err)
	}
	log.Println("ngrok not found in PATH")
	return nil
}

func startNgrokTunnel(localPort string) (string, func(), error) {
	log.Printf("Starting ngrok tunnel on port %s", localPort)
	cmd := exec.Command("ngrok", "http", localPort)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return "", nil, fmt.Errorf("failed to start ngrok tunnel: %w", err)
	}
	cleanup := func() {
		log.Println("Stopping ngrok tunnel...")
		if err := cmd.Process.Kill(); err != nil {
			log.Printf("Failed to kill ngrok tunnel process: %v", err)
		} else {
			log.Printf("ngrok tunnel stopped.")
		}
	}

	ngrokURL := ""
	timeOut := time.After(30 * time.Second)
	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()

	for {
		select {
		case <-timeOut:
			cleanup()
			return "", cleanup, fmt.Errorf("timed out waiting for ngrok tunnel to become active")
		case <-tick.C:
			url, err := git.GetNgrokPublicURL()
			if err == nil && url != "" {
				ngrokURL = url
				log.Printf("Ngrok Public URL obtained: %s", ngrokURL)
				return ngrokURL, cleanup, nil
			}
			log.Println("Waiting for ngrok tunnel to become active...")
		}
	}
}

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
			{
				Name:  "start",
				Usage: "Starts the webhook listener, ngrok tunnel, and optionally sets up Github webhook.",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "repo",
						Usage: "Optional: GitHub repository in the format 'owner/repo-name' (e.g., 'myorg/myproject')",
					},
					&cli.StringFlag{
						Name:    "token",
						Usage:   "Optional: GitHub Personal Access Token with 'repo:hooks' scope",
						EnvVars: []string{"GITHUB_TOKEN"},
					},
				},
				Action: func(c *cli.Context) error {
					if err := ensureNgrokInstalled(); err != nil {
						return err
					}

					ngrokPublicURL, ngrokCleanup, err := startNgrokTunnel(fmt.Sprintf("%d", webhookListenerPort))
					if err != nil {
						return err
					}
					defer ngrokCleanup()

					go func() {
						log.Println("Starting webhook listener...")
						if err := git.StartWebhookListener(); err != nil {
							log.Fatalf("Fatal: Failed to start webhook listener: %v", err)
						}
					}()

					go func() {
						if err := web.StartWebServer(); err != nil {
							log.Fatalf("Fatal: Failed to start web server: %v", err)
						}
					}()

					repoToSetup := c.String("repo")
					tokenToUse := c.String("token")

					if repoToSetup != "" {
						if tokenToUse == "" {
							return cli.Exit("Error: --token is required when --repo is specified for automatic webhook setup", 1)
						}
						log.Printf("Attempting automatic webhook setup for %s...", repoToSetup)
						if err := git.SetupGitHubWebhook(repoToSetup, tokenToUse); err != nil {
							log.Printf("Warning: Failed to setup webhook for %s: %v", repoToSetup, err)
							log.Printf("You might need to manually set up a webhook for %s using `./snap-ci webhook setup` or ensure your Github PAT has necessary permissions", repoToSetup)
						} else {
							log.Printf("Successfully setup webhook for %s", repoToSetup)
						}
					} else {
						log.Println("Skipping automatic webhook setup. Use `./snap-ci webhook setup` to set up a webhook manually")
					}
					fmt.Printf("\nSnapCI is running. Webhook listener is listening on port %s/webhook\n", ngrokPublicURL)
					fmt.Println("Press CTRL+C to stop SnapCI")

					sigchan := make(chan os.Signal, 1)
					signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)
					<-sigchan

					log.Println("Shutting down SnapCI...")
					return nil
				},
			},
			{
				Name:  "trigger",
				Usage: "Manually trigger a run for specific repository. ",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "repo",
						Usage:    "GitHub repository in the format 'owner/repo-name' (e.g., 'myorg/myproject')",
						Required: true,
					},
					&cli.StringFlag{
						Name:  "branch",
						Usage: "Git branch to trigger the run on (e.g., 'main', 'develop', etc.)",
						Value: "main", // Default to 'main'
					},
					&cli.StringFlag{
						Name:  "commit",
						Usage: "Git commit SHA to trigger the run on (Optional)",
					},
				},
				Action: func(c *cli.Context) error {
					repoName := c.String("repo")
					branch := c.String("branch")
					commitSHA := c.String("commit")

					log.Printf("Manually triggering run for repo: %s, branch: %s, commit: %s\n", repoName, branch, commitSHA)
					if err := git.TriggerManualRun(repoName, branch, commitSHA); err != nil {
						return fmt.Errorf("failed to trigger run: %w", err)
					}
					fmt.Printf("Run triggered for repo: %s, branch: %s, commit: %s\n", repoName, branch, commitSHA)
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
