# 🦫 WildPulse Backend (Go / Golang Monorepo)

O **WildPulse** é o monorepo backend de alta performance desenvolvido em Go (Golang 1.26), responsável pela ingestão concorrente de dados de biodiversidade (GBIF, IUCN), geoprocessamento espacial com PostGIS e servimento da API REST para a plataforma WildPulse.

---

## 🏛️ Arquitetura do Monorepo

O projeto utiliza **Go Workspaces (`go.work`)** para gerenciar os serviços da API REST e dos Workers de ingestão de forma modular:

```text
wildpulse/                          (Repositório Backend em Go)
├── go.work                         (Go Workspace file)
├── docker-compose.yml              (PostgreSQL/PostGIS local para dev)
├── scripts/
│   └── init.sql                    (Schema PostGIS e Triggers)
├── apps/
│   ├── api/                        (REST API Server com Chi)
│   │   ├── go.mod
│   │   ├── cmd/server/main.go
│   │   └── internal/
│   │       ├── handler/            (HTTP Endpoints)
│   │       ├── service/            (Camada de Negócio)
│   │       └── repository/         (pgx v5 / Fallback em memória)
│   └── workers/                    (Workers Concorrentes de Ingestão)
│       ├── go.mod
│       ├── cmd/worker/main.go
│       └── internal/
│           ├── collector/          (Cliente HTTP GBIF com Worker Pool)
│           └── enricher/           (Enriquecedor de Status de Ameaça IUCN)
└── pkg/
    ├── domain/                     (Modelos e Structs Go compartilhados)
    │   └── observation.go
    └── spatial/                    (Utilitários de Geoprocessamento)
        └── spatial.go
```

---

## 🛠️ Stack Tecnológica

| Componente | Tecnologia / Biblioteca |
|---|---|
| **Linguagem** | Go 1.26 (Última versão oficial) |
| **HTTP Framework (API)** | Chi v5 (`github.com/go-chi/chi/v5`) |
| **Banco de Dados / Driver** | PostgreSQL 15 + PostGIS via `pgx/v5` |
| **Ingestão Concorrente** | Goroutines + Worker Pools (API GBIF) |
| **Agendador de Tarefas** | Cron v3 (`github.com/robfig/cron/v3`) |

---

## 🚀 Como Rodar Localmente

### 1. Iniciar o Banco de Dados PostGIS (Docker)
```bash
docker compose up -d
```

### 2. Rodar a API REST (Server)
```bash
go run ./apps/api/cmd/server/main.go
```
> Servidor HTTP ativo em: `http://localhost:8080`

### 3. Rodar os Workers de Ingestão
```bash
go run ./apps/workers/cmd/worker/main.go
```

---

## 📡 API REST Endpoints

| Método | Endpoint | Descrição |
|---|---|---|
| `GET` | `/health` | Checagem de saúde da aplicação |
| `GET` | `/api/v1/observations` | Lista observações com suporte a filtros (`min_lat`, `max_lat`, `min_lng`, `max_lng`, `biome`, `taxa`, `limit`, `offset`) |
| `GET` | `/api/v1/species/{id}` | Detalhes de uma espécie e seu histórico de ocorrências |
| `GET` | `/api/v1/stats` | Resumo de métricas da plataforma e biomas |

---

## 🌐 Hospedagem 100% Gratuita

1. **API Server & Workers**: Render (Free Tier), Fly.io ou Koyeb.
2. **Banco Spacial PostGIS**: Neon.tech (PostgreSQL Free Tier) ou Supabase.
