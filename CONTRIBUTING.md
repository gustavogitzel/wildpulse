# 🤝 Contributing to WildPulse Backend

Thank you for your interest in contributing to **WildPulse**! This guide outlines the workflow and standards for developing on the Go backend monorepo.

---

## 🛠️ Development Setup

### Prerequisites
- **Go**: 1.26 or later (`go version`)
- **Docker / Colima**: Docker Compose or Colima with PostgreSQL 15 + PostGIS
- **Make**: Standard build automation tool

### Getting Started
1. Clone the repository:
   ```bash
   git clone https://github.com/wildpulse/wildpulse.join
   cd wildpulse
   ```
2. Start the local PostGIS container:
   ```bash
   make docker-up
   ```
3. Run the REST API:
   ```bash
   make run-api
   ```
4. Run unit tests to verify your environment:
   ```bash
   make test
   ```

---

## 📁 Repository Structure

We use **Go Modules Workspaces (`go.work`)**:
- `pkg/`: Shared domain models (`domain`) and spatial helpers (`spatial`).
- `apps/api/`: REST API server powered by Chi router.
- `apps/workers/`: Data ingestion worker routines and cron schedules.

---

## 📋 Code Guidelines

1. **Go Idioms & Formatting**:
   - Run `go fmt ./...` before committing.
   - Write clear Go doc comments for all exported structs, interfaces, and functions.
2. **Testing**:
   - Add unit tests for all new domain models, spatial logic, or API handlers.
   - Run `make test` to ensure all tests pass.
3. **Commit Messages**:
   - Follow Conventional Commits convention:
     - `feat: add new endpoint for biome analytics`
     - `fix: resolve bounding box lat/lng validation edge case`
     - `docs: update OpenAPI specification and Makefile`

---

## 📬 Submitting Pull Requests

1. Fork the repo and create your feature branch: `git checkout -b feat/my-awesome-feature`.
2. Ensure `make build` and `make test` complete without errors.
3. Push your branch and open a Pull Request describing your changes.
