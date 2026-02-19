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
- **Angular 21** with **Material Design**
- **Signal-based** state everywhere (except `httpResource`)
- **Signal Forms**
- Served embedded in the Go backend binary

### PostgreSQL
- **PG 18+** for native UUIDv7 support
- Strong data consistency enforced at database level

### RustFS Object Storage
- S3-compatible API (MinIO replacement after license change)

### Ollama LLM Service
- OpenAI-compatible API
- Configurable model

### Tesseract OCR Service
- Custom Go web server using gosseract
- OpenOCR-compatible API

## Service Communication
- Synchronous REST between all services
- Jobs exposed via REST endpoints, triggered by cron container ([docker-compose cron pattern](https://distr.sh/blog/docker-compose-cron-jobs/))

## Open Decision: Event Sourcing
- **Option A**: Full CQRS with [watermill](https://github.com/ThreeDotsLabs/watermill)
- **Option B**: Simple audit log table (already designed in schema)
