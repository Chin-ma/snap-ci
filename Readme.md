# SnapCI: A Lightweight, Self-Hosted CI/CD Tool

**SnapCI** is a minimalist Continuous Integration/Continuous Delivery tool written in Golang. It's designed for simplicity and minimal setup—ideal for local development or small teams needing Git-based pipelines without the overhead of larger CI/CD systems.

---

## ✨ Features

- **Git-based Configuration**: Pipelines defined in a `.ci.yaml` file in your repository.
- **Git Webhook Listener**: Automatically triggers pipelines on GitHub push events.
- **Automated Webhook Setup**: CLI and Web UI commands to configure GitHub webhooks using dynamic ngrok URLs.
- **Private Repo Auth**: Secure storage of GitHub Personal Access Tokens (PATs) for cloning private repos.
- **Local Logs & Run History**: Stores detailed logs and metadata locally.
- **Simple Web Dashboard**: View run history, manage webhooks, and auth via a basic UI.
- **Single Binary**: Easily deployable as a standalone executable.

---

## 🚀 Getting Started (For Developers)

### Prerequisites

Ensure the following tools are installed:

- **Go (1.16+)**  
  [Download Go](https://golang.org/dl/)

- **Git CLI**  
  [Download Git](https://git-scm.com/downloads)

- **ngrok**  
  [Download ngrok](https://ngrok.com/download)  
  After install:  
  ```bash
  ngrok config add-authtoken <YOUR_NGROK_AUTH_TOKEN>


* **GitHub Personal Access Token (PAT)**
  Go to: `GitHub Settings > Developer settings > Personal access tokens > Tokens (classic)`
  Generate a new token with:

  * `admin:repo_hook` (for webhook setup)
  * `repo` (for cloning private repositories)

> ⚠️ Save your PAT securely. It won't be visible again after creation.

---

### 🔧 Build the Application

```bash
go build -o snapci .
```

This generates a `snapci` executable in the current directory.

---

## 🛠️ Usage

### 1. Fully Automated Start (Recommended)

```bash
./snapci start --repo <owner/repo-name> --token <your_github_pat>
```

Or using environment variable:

```bash
export GITHUB_TOKEN="your_github_pat"
./snapci start --repo your-org/your-repo
```

Once running:

* **Webhook Listener**: `https://<ngrok-url>/webhook`
* **Web Dashboard**: `http://localhost:8081`

Push to your GitHub repo to trigger pipelines. Use `Ctrl+C` to stop.

---

### 2. CLI Commands

#### Run Pipeline Manually

```bash
./snapci run --config .ci.yaml
```

#### Start Webhook Listener Only

```bash
./snapci webhooks
# Then in another terminal:
ngrok http 8080
```

#### Setup GitHub Webhook

```bash
./snapci webhook setup --repo <owner/repo-name> --token <your_github_pat>
# Or use env variable:
export GITHUB_TOKEN="your_github_pat"
./snapci webhook setup --repo <owner/repo-name>
```

#### Add GitHub PAT for Repo

```bash
./snapci auth add --repo <owner/repo-name> --token <your_github_pat>
# Or with env:
export GITHUB_PAT="your_github_pat"
./snapci auth add --repo <owner/repo-name>
```

> 🔒 **Security Warning**: Tokens are stored in `./auth_data/` as plaintext JSON. Restrict file access or use a secrets manager for production.

#### View Logs

```bash
./snapci logs --id <run-id>
```

#### View Run Status (WIP)

```bash
./snapci status [--id <run-id> | --recent]
```

#### Start Only Web UI

```bash
./snapci web
```

---

### 3. Web Dashboard

Access: [http://localhost:8081](http://localhost:8081)

Features:

* **Run History**: View all pipeline runs, statuses, and logs.
* **Add Repo Auth**: `/add-auth` to store PATs for private repos.
* **Setup Webhooks**: `/setup-webhook` for GitHub webhook integration.

---

## ⚙️ Pipeline Configuration (`.ci.yaml`)

SnapCI uses `.ci.yaml` in the root of your Git repo to define jobs and steps.

### Example:

```yaml
name: build-and-test
on: [push]

jobs:
  build:
    steps:
      - name: Install Dependencies
        run: npm install
      - name: Run Tests
        run: npm test
      - name: Build Application
        run: npm run build

  deploy:
    needs: build
    steps:
      - name: Deploy to Staging
        run: echo "Deploying to staging..."
```

---

## 🔒 Security Considerations

* **GitHub PATs**: Treat them as passwords. Avoid committing or exposing them.
* **ngrok**: Exposes your local machine to the internet—run only trusted services during active tunnels.

---

## 🤝 Contribution

Have feedback, found a bug, or want to add a feature?
Feel free to open an issue or submit a pull request on GitHub!

---