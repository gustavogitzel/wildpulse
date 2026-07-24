# 🦫 WildPulse Backend (Go / Golang Monorepo)

O **WildPulse** é o monorepo backend de alta performance desenvolvido em Go (Golang 1.26), responsável pela ingestão concorrente de dados de biodiversidade (GBIF, IUCN), geoprocessamento espacial com PostGIS no Neon.tech e servimento da API REST para a plataforma WildPulse.

---

## 🏛️ Arquitetura do Monorepo

O projeto utiliza **Go Workspaces (`go.work`)** para gerenciar os serviços da API REST e dos Workers de ingestão de forma modular:

```text
wildpulse/                          (Repositório Backend em Go 1.26)
├── go.work                         (Go Workspace file)
├── Makefile                        (Comandos de build, test e run)
├── render.yaml                     (Blueprint de Deploy no Render Cloud)
├── docker-compose.yml              (PostgreSQL/PostGIS local para dev)
├── migrations/                     (Migrações SQL embedded com Go embed)
│   └── 000001_init_schema.up.sql
├── apps/
│   ├── api/                        (REST API Server com Chi v5)
│   │   ├── go.mod
│   │   ├── cmd/server/main.go
│   │   └── internal/
│   │       ├── handler/            (HTTP Handlers + OpenAPI Swagger embedded)
│   │       ├── service/            (Camada de Negócio e Disparo de Ingestão)
│   │       └── repository/         (Consultas Diretas em SQL no PostgreSQL PostGIS)
│   └── workers/                    (Workers Concorrentes de Ingestão Cron)
│       ├── go.mod
│       └── cmd/worker/main.go
└── pkg/
    ├── domain/                     (Modelos e Structs Go compartilhados)
    ├── database/                   (Motor de Migrações Nativas Embedded)
    ├── collector/                  (Cliente HTTP GBIF com Worker Pool Concorrente)
    ├── enricher/                   (Enriquecedor de Status de Ameaça IUCN)
    └── spatial/                    (Utilitários de Geoprocessamento PostGIS)
```

---

## 🛠️ Stack Tecnológica

| Componente | Tecnologia / Biblioteca |
|---|---|
| **Linguagem** | Go 1.26 |
| **HTTP Framework (API)** | Chi v5 (`github.com/go-chi/chi/v5`) |
| **Banco de Dados / Driver** | PostgreSQL 15 + PostGIS via `pgxpool v5` (Neon.tech Cloud) |
| **Documentação Interativa** | OpenAPI 3.0 + Swagger UI (`/swagger` e `/docs`) |
| **Ingestão Concorrente** | Goroutines + Worker Pools (API GBIF & IUCN) |
| **Agendador de Tarefas** | Cron v3 (`github.com/robfig/cron/v3`) |

---

## 📡 Endpoints da API REST

| Método | Endpoint | Descrição |
|---|---|---|
| `GET` | `/health` | Healthcheck do serviço |
| `GET` | `/swagger` / `/docs` | Interface gráfica interativa da Documentação Swagger UI |
| `GET` | `/api/v1/observations` | Lista paginada de ocorrências espaciais com filtros (bioma, bounding box, busca) |
| `POST` | `/api/v1/observations/trigger` | **Disparo sob demanda** do pipeline de ingestão concorrente GBIF + IUCN |
| `GET` | `/api/v1/species/{id}` | Detalhes taxonômicos e histórico de avistamentos de uma espécie |
| `GET` | `/api/v1/stats` | Agregação de métricas da plataforma e distribuição por biomas |

---

## 🚀 Como Rodar Localmente

### 1. Iniciar o Servidor API REST
```bash
make run-api
```
> Documentação OpenAPI Swagger disponível em: `http://localhost:8080/swagger`

### 2. Rodar o Worker de Ingestão
```bash
make run-worker
```

### 3. Rodar os Testes Unitários
```bash
make test
```

### 4. Compilar os Binários de Produção
```bash
make build
```
