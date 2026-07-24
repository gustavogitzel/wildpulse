# 📜 Changelog

All notable changes to the **WildPulse Backend** monorepo will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [1.1.0] - 2026-07-24

### ✨ Added
- **On-Demand Ingestion Endpoint**: Added `POST /api/v1/observations/trigger` endpoint to execute the GBIF occurrence collector & IUCN threat enrichment pipeline on-demand, saving records directly to PostgreSQL PostGIS.
- **OpenAPI 3.0 Updates**: Updated embedded `swagger.json` documentation and Swagger UI (`/swagger`, `/docs`) with the new trigger endpoint specification.

### 🔄 Changed
- **Shared Ingestion Modules**: Refactored `collector` and `enricher` from internal worker packages to `pkg/collector` and `pkg/enricher`, enabling clean shared access between worker cron jobs and API trigger handlers.

### 🧹 Removed
- **Mock Data Fallbacks**: Removed all mock data in-memory fallbacks from `PostgresRepository`, ensuring 100% of API endpoints interact directly with PostgreSQL PostGIS database tables (`observations`, `species`).

---

## [1.0.0] - 2026-07-24

### ✨ Added
- **Monorepo Architecture**: Setup Go 1.26 workspace (`go.work`) with `./pkg`, `./apps/api`, and `./apps/workers`.
- **Domain Layer (`pkg/domain`)**: Core data models for `Observation`, `Species`, `IUCNStatus` enum (`CR`, `EN`, `VU`, `NT`, `LC`), `PaginatedResult`, and `StatsSummary`.
- **Geospatial Utilities (`pkg/spatial`)**: Bounding Box (`BBox`), Haversine distance calculations, PostGIS query builders (`ST_MakeEnvelope`), and South American biome detection algorithm (`DetectBiome`).
- **REST API Server (`apps/api`)**: High-performance HTTP REST server powered by **Chi v5**, featuring:
  - `GET /health`: Service health status.
  - `GET /api/v1/observations`: Bounding box, biome, taxa, and status filtering.
  - `GET /api/v1/species/{id}`: Species metadata & occurrence history.
  - `GET /api/v1/stats`: Aggregated biodiversity platform statistics.
  - `pgx/v5` PostgreSQL/PostGIS integration with fallback in-memory dataset store.
- **Ingestion Workers (`apps/workers`)**:
  - **GBIF Collector**: Concurrent HTTP ingestion client using Goroutines & Worker Pools fetching occurrence media records across South America.
  - **IUCN Enricher**: Species conservation status threat enrichment worker.
  - **Cron Scheduler**: Background execution via `robfig/cron/v3` every 30 minutes.
- **OpenAPI / Swagger Documentation**: Embedded OpenAPI 3.0 specification (`swagger.json`) served via Swagger UI at `/swagger` and `/docs`.
- **Automated Database Migrations (`pkg/database`)**: Built-in `embed` database migration runner executing `.up.sql` scripts sequentially inside database transactions upon API startup.
- **Containerization & Cloud Infrastructure**:
  - Docker Compose setup with `postgis/postgis:15-3.3-alpine`.
  - Multi-stage Dockerfiles (`apps/api/Dockerfile` and `apps/workers/Dockerfile`) producing minimal ~15MB runtime images.
  - Render Blueprint (`render.yaml`) for 1-click cloud deployment of Web API and Background Workers.
- **GitHub Actions CI/CD Pipeline (`.github/workflows`)**:
  - `ci.yml`: Automated build, Go workspace verification, and unit/integration testing on push/PR.
  - `deploy.yml`: Render deployment trigger integration.
- **Developer Automation**:
  - Comprehensive `Makefile` with Colima Docker socket auto-detection.
  - Community documentation suite (`README.md`, `CONTRIBUTING.md`, `LICENSE`, `CHANGELOG.md`, `.gitignore`).
