# Tech Stack

## Monorepo Structure

- Single repo: backend, frontend, docs, mobile apps
- Frontend embedded into Go binary as static files

## Build Tools

- **Mise** — tool version management and task runner
- **pnpm** — frontend/docs package management
- **ko** — Go container image building (distroless)

## Services

### Go Backend

- **OpenAPI spec-first** — routes and DTOs generated with [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen) using Chi router
- **sqlc** with **pgx** — type-safe generated CRUD queries ([sqlc](https://github.com/sqlc-dev/sqlc))
- **go-migrate** — in-application migration management
- Hand-written business logic and mapping layer

### Angular Frontend

- **Angular** with `@angular/material`
- **Signal-based** state everywhere (except `httpResource`)
- **Signal Forms**
- Served by the Go backend binary using `http.FileServer` and `embed.FS`

### PostgreSQL

- **PG 18+** for native UUIDv7 support
- Strong data consistency enforced at database level

### Object Storage

- **S3-compatible API** for document/file storage

### LLM Service

- **OpenAI-compatible API** for AI features

### OCR Service

- **Tesseract** via REST API

## Service Communication

- Synchronous REST between all services
- Jobs exposed as CLI commands, scheduled via cron
