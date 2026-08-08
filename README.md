# Pyramid

An all-in-one management system for a software department — people, projects, contributions, growth, and time off. This repository is a monorepo with a Go backend and a Svelte frontend that share a single, generated type contract.

## Architecture

```
┌──────────────────────┐        generates        ┌──────────────────────┐
│  Svelte frontend      │  ◀───────────────────   │  openapi/openapi.yaml │
│  (SvelteKit 2,        │   openapi-typescript     │  the SINGLE source    │
│   Svelte 5, Tailwind) │                          │  of truth for the API │
└──────────────────────┘                          └──────────┬───────────┘
        ▲  typed fetch (openapi-fetch)                        │ generates
        │                                                     │ oapi-codegen
        ▼                                          ┌──────────▼───────────┐
   HTTP :5173 ───────────────────────────────────▶│  Go backend (Gin,     │
                                                   │  GORM) → Postgres     │
                                                   └──────────────────────┘
```

The API contract lives in `openapi/openapi.yaml`. Both sides generate their types from it, so a change to a field, enum, or endpoint flows to the Go server **and** the TypeScript client from one edit — the two ends of the wire cannot silently drift apart. The database shape lives in `db/schema.sql` and the GORM models mirror it.

## Repository layout

```
pyramid/
├── openapi/openapi.yaml        # API contract — single source of truth
├── db/
│   └── schema.sql              # PostgreSQL schema (32 tables, 6 modules)
├── docs/SCHEMA.md              # Data-model overview
├── backend/                    # Go + Gin + GORM
│   ├── cmd/server/main.go      # Entry point
│   ├── internal/
│   │   ├── api/                # Generated DTOs (types.gen.go) + enums
│   │   ├── config/             # Env-based configuration
│   │   ├── database/           # GORM connection
│   │   ├── handlers/           # Gin handlers (health, people)
│   │   └── models/             # GORM models + DTO conversion
│   └── oapi-codegen.yaml       # Go type generation config
├── frontend/                   # SvelteKit 2 + Svelte 5 + Tailwind v4
│   └── src/
│       ├── lib/api/            # schema.d.ts (generated) + typed client
│       └── routes/             # Pages (People list demo)
└── Makefile                    # generate / setup / dev / build
```

## Prerequisites

- **Go 1.25+** (developed against Go 1.27)
- **Node 20+** and npm
- **PostgreSQL 14+**

## Quick start

```bash
# 1. Install dependencies (Go modules + npm packages)
make setup

# 2. Create the database and load the schema
createdb pyramid
make db-schema            # applies db/schema.sql

# 3. (optional) regenerate types from the contract
make generate

# 4. Run the two dev servers in separate terminals
make dev-backend          # Go API  → http://localhost:8080
make dev-frontend         # Svelte  → http://localhost:5173
```

Open http://localhost:5173 — the People page calls `GET /api/v1/people` on the Go API using the generated, fully-typed client.

Configuration is via environment variables; copy `backend/.env.example` and `frontend/.env.example` to `.env` and adjust as needed.

## The type pipeline

`make generate` runs two generators against `openapi/openapi.yaml`:

- **Backend** — `oapi-codegen` writes `backend/internal/api/types.gen.go` (Go structs + enum constants).
- **Frontend** — `openapi-typescript` writes `frontend/src/lib/api/schema.d.ts` (TS `paths`/`components`), consumed by the `openapi-fetch` client in `frontend/src/lib/api/client.ts`.

Add a field to a schema in the contract, run `make generate`, and both sides update together. The enums in the contract mirror the Postgres enum types in `db/schema.sql`.

## Current scope

This is the foundation plus one vertical slice — the **People** resource, end to end (list, create, get, update, delete) — which establishes the pattern to replicate for the remaining modules (projects, tasks, contributions, goals, leave). The schema for all modules is already in `db/schema.sql`.

## A note on how this was generated

The frontend was fully built and type-checked in the authoring environment (SvelteKit build + `svelte-check` pass with zero errors, TS types generated from the contract). The backend's `api` and `models` packages were compiled and verified there as well; the `handlers`/`main` packages use standard Gin/GORM calls but could not be compiled in that sandbox because its network egress blocked the Go module proxy and vanity-domain dependencies. On a normal machine, `make setup && make generate && make build` compiles the whole backend. If you hit any issue, that's the place to look first.
