## 📌 Objective

Develop a lightweight, self-hosted CI/CD pipeline tool written in **Golang**, designed for simplicity and minimal setup. The tool should support:
- Git-based configuration (`.ci.yaml`)
- Docker-based step execution
- Minimal system dependencies
- Local or small-team use cases

---

## 🧑‍💻 Target Audience

- DevOps engineers and solo developers
- Teams practicing GitOps
- Users needing a self-hosted, lightweight CI/CD solution
- Developers looking for Docker-native pipelines with Git-only config

---

## 🎯 Goals

### ✅ Must-Have Features (MVP)
- Git webhook listener (GitHub/GitLab)
- Git-based pipeline configuration via `.ci.yaml`
- Step execution in Docker containers
- CLI interface for pipeline status and logs
- Local storage for logs and run history

### 🌟 Nice-to-Have Features
- Minimal web UI to view pipeline runs and logs
- Build status badges for repositories
- Slack/email notifications
- Retry mechanism for failed jobs
- Secure secret management (for env vars)
- Authentication (basic or token-based)

---

## 🧩 Core Features Description

### 1. Git Webhook Listener
- Handles push events
- Parses webhook payload
- Verifies webhook signature (HMAC)
- Triggers pipeline execution if `.ci.yaml` is found

### 2. YAML-Based Pipeline Configuration

```yaml
name: build-and-deploy

on: [push]

jobs:
  build:
    steps:
      - name: Install dependencies
        run: go mod tidy
      - name: Run tests
        run: go test ./...

  deploy:
    needs: build
    steps:
      - name: Docker build
        run: docker build -t myapp .
      - name: Docker push
        run: docker push myapp
```

### 3. Pipeline Executor

* Clones the repository
* Parses `.ci.yaml` into job steps
* Executes each step in an isolated Docker container
* Captures logs and status of each step

### 4. CLI Interface

* View job status and logs
* Filter jobs by repo, commit, branch
* Retry or cancel a job

### 5. (Optional) Web Dashboard

* Displays list of recent jobs
* Visual indicator of job status (Success, Failure, Running)
* Live log view using WebSockets (optional)

---

## 🧱 Architecture

```
ci-pipeline-go/
├── cmd/                 # CLI/Web entry point
├── config/              # YAML config parsing
├── executor/            # Pipeline job runner
├── git/                 # Git webhook handling
├── pipeline/            # Orchestrator and scheduler
├── storage/             # Logs and job metadata
├── web/                 # (Optional) Web UI server
├── go.mod / go.sum
└── README.md
```

---

## 🔧 Tech Stack

* **Language**: Go
* **Execution**: Docker SDK for Go
* **Webhook Server**: net/http
* **Config**: YAML parsing via `gopkg.in/yaml.v3`
* **Storage**: File-based logs or SQLite/BadgerDB
* **CLI**: Cobra or urfave/cli (TBD)
* **Web UI**: Go templates or React (optional)

---

## 🔄 Pipeline Execution Flow

1. Push is made to the Git repository.
2. GitHub/GitLab sends a webhook.
3. GoCI receives the webhook and verifies it.
4. The repository is cloned into a temporary directory.
5. `.ci.yaml` is parsed and jobs are constructed.
6. Each job step runs inside a Docker container.
7. Logs are streamed and saved.
8. CLI or Web UI shows job status.

---

## 🚀 Development Phases

### Phase 1: Core System

* [ ] Webhook listener for GitHub
* [ ] YAML parser for `.ci.yaml`
* [ ] Job runner using `os/exec` (temporary)
* [ ] CLI to check status/logs

### Phase 2: Docker Execution

* [ ] Replace `os/exec` with Docker SDK
* [ ] Handle job isolation via temp directories

### Phase 3: Web Dashboard (Optional)

* [ ] UI to list jobs and view logs
* [ ] Add WebSocket-based log streaming

### Phase 4: Enhancements

* [ ] Retry failed steps
* [ ] Add build status badge support
* [ ] Add secrets management
* [ ] Webhook signature verification
* [ ] Notifications (Slack/email)

---

## 🧠 Competitive Analysis

| Tool           | Language | Self-Hosted | Docker Support | Git-Only Config | Complexity |
| -------------- | -------- | ----------- | -------------- | --------------- | ---------- |
| Jenkins        | Java     | ✅           | ⚠️ Partial     | ❌               | High       |
| Drone CI       | Go       | ✅           | ✅              | ✅               | Medium     |
| GitHub Actions | JS       | ❌ (cloud)   | ✅              | ✅               | Medium     |
| GoCI (This)    | Go       | ✅           | ✅              | ✅               | Low        |

---

## 📦 Deliverables

* Go-based CLI binary
* Git webhook listener
* Docker step executor
* Example `.ci.yaml` pipelines
* Local runner logs
* (Optional) Web dashboard
* Docker image for runner

---

## 📅 Suggested Timeline

| Week | Tasks                              |
| ---- | ---------------------------------- |
| 1    | Webhook listener + Git clone logic |
| 2    | YAML parser + Local step execution |
| 3    | Docker step runner + log storage   |
| 4    | CLI interface + MVP complete       |
| 5    | (Optional) Web UI for status/logs  |
| 6    | Testing, cleanup, docs, release    |

---

## 🚫 Out of Scope (for MVP)

* Distributed builds or agent-based execution
* OAuth or multi-user login
* Plugin/extension system
* Kubernetes integration
* Cron-based pipelines

---

## 📘 Resources & Inspiration

* [Drone CI](https://github.com/harness/drone)
* [Woodpecker CI](https://github.com/woodpecker-ci/woodpecker)
* [Act](https://github.com/nektos/act)
* [Docker SDK for Go](https://pkg.go.dev/github.com/docker/docker/client)

---

## 🙋 Contact & Contribution

Interested contributors and testers are welcome. Create issues for bugs and feature requests. PRs should be accompanied by tests and clear commit messages.

---

```

Let me know if you want:
- This PRD in a downloadable `.md` file.
- A GitHub repository scaffold for this project.
- Help writing the first modules (e.g. webhook listener or Docker executor).
```
````markdown
