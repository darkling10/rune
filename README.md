# Rune ᛤ

**Rune** is an AI-native Continuous Integration and Continuous Deployment (CI/CD) engine. It acts as an intelligent gatekeeper for your infrastructure, seamlessly integrating standard shell execution with autonomous LLM code reviews to block vulnerabilities before they reach production.

![Rune Control Plane](web/public/logo.jpg)

## Features

- **Dynamic YAML Engine:** Configure your pipelines using a simple, declarative `.rune.yaml` file (similar to GitHub Actions).
- **AI Gatekeeper (`rune/ai-review@v1`):** Natively supports halting the pipeline if an AI (OpenAI, Claude, Bedrock) detects critical security vulnerabilities or logic flaws in the `git diff`.
- **Asynchronous Worker Architecture:** Built on top of Redis and Asynq for robust, scalable, and background task processing.
- **GitOps Ready:** Natively integrates with GitHub webhooks to trigger pipeline runs instantly on push.
- **React Control Plane:** A beautiful, responsive Vite + React + Tailwind v4 dashboard to monitor pipelines and securely manage API keys.

## Architecture

Rune is built as a highly concurrent Go application using a true monorepo structure:

- `cmd/server`: The main entry point that boots the API Server and the Worker daemon.
- `internal/api`: The REST API layer (built with `chi`) providing data to the React UI and ingesting GitHub Webhooks.
- `internal/worker`: The Asynq-powered background job processing engine that clones repos and orchestrates pipelines.
- `pkg/pipeline`: The execution engine that parses `.rune.yaml` and streams shell commands (or AI actions) safely.
- `web/`: The React 19 Control Plane UI.

## Getting Started

### Prerequisites
- Go 1.22+
- Node.js (for the React UI)
- Docker & Docker Compose (for PostgreSQL, Redis, and Floci)

### 1. Spin up the Infrastructure
Rune relies on PostgreSQL for persistence, Redis for the worker queues, and Floci for local AWS emulation.
```bash
docker compose up -d
```

### 2. Start the Rune Server
```bash
go run ./cmd/server/main.go
```

### 3. Start the Control Plane UI
```bash
cd web
npm install
npm run dev
```
Navigate to `http://localhost:5173` to view the dashboard!

## Creating your first `.rune.yaml`

To enable Rune on your project, just commit a `.rune.yaml` to the root of your repository:

```yaml
version: "1.0"
pipeline:
  - name: "Build Artifact"
    run: "docker"
    args: ["build", "-t", "my-app:latest", "."]

  # The AI Gatekeeper
  - name: "Security Review"
    uses: "rune/ai-review@v1"
    
  - name: "Push to Production"
    run: "docker"
    args: ["push", "my-app:latest"]
```
