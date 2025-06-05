## Installation

1.  **Download SnapCI:**
    Download the appropriate binary for your operating system and architecture from the [GitHub Releases page](LINK_TO_YOUR_GITHUB_RELEASES).

2.  **External Dependencies:**
    SnapCI relies on the following tools being installed and available in your system's PATH:
    * **Git CLI:** Required for cloning repositories.
        (Install from: [https://git-scm.com/downloads](https://git-scm.com/downloads))
    * **Docker:** Required for executing pipeline steps in containers. Ensure the Docker daemon is running.
        (Install from: [https://www.docker.com/get-started](https://www.docker.com/get-started))
    * **ngrok:** Used to expose your local webhook listener to the internet.
        (Install from: [https://ngrok.com/download](https://ngrok.com/download))
        * **Important:** After installing ngrok, you'll need to authenticate it once with your ngrok auth token: `ngrok config add-authtoken <your_ngrok_auth_token>`

3.  **Make Executable (Linux/macOS):**
    After downloading, make the binary executable:
    ```bash
    chmod +x ./snapci-linux-amd64 # Or your specific binary name
    ```

4.  **Run SnapCI:**
    You can now run SnapCI! For the fully automated experience:
    ```bash
    ./snapci start --repo your-github-org/your-repo --token your_github_pat
    ```
    (Replace with your actual GitHub info. For security, consider using `GITHUB_TOKEN` environment variable instead of `--token` flag.)

    SnapCI will start ngrok, the webhook listener, the web dashboard, and optionally set up your GitHub webhook.
    * Webhook listener: `https://<ngrok-public-url>/webhook`
    * Web dashboard: `http://localhost:8081`