# Persistent Agent Context

## General

- When fetching information from GitHub, always use the GitHub CLI (`gh`) command-line tool

## Project Structure

```
api/                  # OpenAPI specification (YAML, source of truth)
internal/
├── cmd/              # CLI commands (serve, migrate)
├── aggregate/        # Domain aggregates and repositories
├── events/           # Domain events, event store, serialization
├── projections/      # Read model projection handlers
├── datei/            # Domain services (business logic orchestration)
├── db/               # sqlc config, SQL queries, generated code, migrations
├── server/           # HTTP endpoints (oapi-codegen generated + handlers: endpoints_auth, endpoints_settings, endpoints_emails, endpoints_datei)
├── storage/          # File storage abstraction (S3)
├── mapping/          # DTO conversions (aggregate/projection → API)
├── config/           # Viper-based configuration
├── dateierrors/      # Domain sentinel errors
├── frontend/         # Static frontend serving
└── buildconfig/      # Build metadata
pkg/api/              # Generated API models from OpenAPI spec
frontend/
├── src/
│   ├── api/          # Generated TypeScript client from OpenAPI
│   ├── app/          # Angular application
│   └── ...
```

## Development Workflow

- Start dev services: `podman compose up -d` (PostgreSQL, Rustfs S3, Mailpit)
- Run server: `mise serve` (alias for `mise run:datei:serve`)
- Run tests: `mise run test`
- Lint: `mise run lint`
- Always run `mise run build:datei` and `mise run format` after making changes

## Code Generation (CRITICAL)

This project relies heavily on code generation. NEVER edit files prefixed with `zz_generated`. Instead, edit the source files and regenerate.

**ALL HTTP endpoints must be defined in the OpenAPI spec first** (`api/paths/*.yaml`), including auth and settings endpoints. Never hand-write Chi/HTTP handlers without an OpenAPI definition. The generated `StrictServerInterface` is the contract — implement its methods in `internal/server/endpoints_*.go`.

### Generation Chain

1. **OpenAPI spec** (`api/**/*.yaml`) → bundled to `dist/openapi.yaml` via Redocly
2. **Backend models** (`dist/openapi.yaml`) → `pkg/api/zz_generated.models.go` via oapi-codegen
3. **Backend server** (`dist/openapi.yaml`) → `internal/server/zz_generated.server.go` via oapi-codegen
4. **Database code** (`internal/db/*.sql` + `zz_generated_schema.sql`) → `internal/db/zz_generated.*.go` via sqlc
5. **Frontend client** (`dist/openapi.yaml`) → `frontend/src/api/` via ng-openapi-gen

Run `mise generate` to regenerate everything (backend + frontend).

## Feature Development Checklist

When implementing a new feature, follow these steps **in order**. Steps are annotated with when they can be skipped.

#### Phase 1: Infrastructure (schema, API spec, code generation)

1. **Add database migration** in `internal/db/migrations/sql/` _(skip if frontend-only)_
   - Create `<next_version>_<name>.up.sql` and `<next_version>_<name>.down.sql`
   - Add/alter tables, columns, indexes, or enums needed for the new feature
   - Run `mise run:datei:migrate` to apply the migration against the running database
   - Run `mise import-db-schema` to export the live schema to `internal/db/zz_generated_schema.sql` — sqlc reads this file, so generation will fail or produce wrong types without it

2. **Define the API endpoint** in the OpenAPI spec _(skip if frontend-only with no new API)_
   - Add/update path in `api/paths/<name>.yaml` and reference it in `api/openapi.yaml`
   - Add request/response schemas in `api/components/schemas/` and `api/components/requestBodies/`

3. **Write the SQL queries** in `internal/db/*.sql` _(skip if frontend-only)_
   - Add projection queries in `internal/db/datei.sql` (or a new `.sql` file)
   - Use sqlc comment format: `-- name: QueryName :exec` or `:one` or `:many`

4. **Run code generation**: `mise generate`
   - This regenerates backend models, server interface, sqlc Go code, AND the frontend TypeScript client
   - Always run this after changing OpenAPI specs, SQL queries, or the database schema

#### Phase 2: Event Sourcing (domain logic) _(skip entirely if frontend-only)_

5. **Define the event** in `internal/events/domain.go`
   - Create a struct implementing `DomainEvent` (with `EventType()` and `StreamID()` methods)
   - Use JSON tags for serialization
   - Include all data needed to reconstruct state AND update projections
   - `EventType()` returns a PascalCase string (e.g., `"DateiRenamed"`)

6. **Add serialization** in `internal/events/serialization.go`
   - Add a case to `Deserialize()` for the new event type string
   - `Serialize()` uses `json.Marshal` and works automatically via the JSON tags

7. **Add the command** to the aggregate in `internal/aggregate/aggregate.go`
   - Validate preconditions (aggregate state, input values)
   - Create the event struct and call `a.recordEvent(event)`
   - `recordEvent` increments version, appends to uncommitted list, and calls `ApplyEvent`

8. **Add state application** in `ApplyEvent()` in `internal/aggregate/aggregate.go`
   - Add a `case` in the type switch to update aggregate fields from event data
   - This must be a pure function of current state + event (no side effects)

9. **Add the projection handler** in `internal/projections/handlers.go`
   - Create `UpdateProjectionFor<EventName>()` using generated sqlc `Queries`
   - Projections run synchronously inside the same transaction as event append

10. **Wire the projection** in `internal/aggregate/repository.go`
    - Add a case in `updateProjection()` to dispatch to the new handler

#### Phase 3: Service and HTTP layer _(skip if frontend-only)_

11. **Add the service method** in `internal/datei/datei.go` (or new service file)
    - Define `Input`/`Output` structs for the operation
    - Load aggregate via `repository.LoadByID()`, call command, call `repository.Save()`
    - For read operations, query projections directly via `db.New(pool)`

12. **Add the HTTP endpoint** in `internal/server/endpoints_<domain>.go`
    - Implement the generated `StrictServerInterface` method
    - Map HTTP request → service input, call service, map output → HTTP response

13. **Add DTO mapping** in `internal/mapping/` if needed

#### Phase 4: Frontend _(skip if backend-only)_

14. **Implement the UI** using the generated API client from `frontend/src/api/`
    - The TypeScript client was already regenerated in step 4
    - Follow the conventions in the Frontend Conventions section below

## Event Sourcing Architecture

This project uses Event Sourcing with synchronous projections.

### Write Path (Command Side)

```
HTTP Request → Server Endpoint → Service → Aggregate Command → Event(s)
                                         → Repository.Save():
                                            1. Begin TX
                                            2. EventStore.AppendToStream (optimistic locking)
                                            3. Update projections (same TX)
                                            4. Commit TX
```

### Read Path (Query Side)

```
HTTP Request → Server Endpoint → Service → db.Queries (read from projection tables)
```

### Key Patterns

- **Optimistic locking**: `AppendToStream` checks `expectedVersion` matches the current stream version before inserting
- **Transactional consistency**: Events and projection updates happen in a single PostgreSQL transaction
- **Aggregate reconstruction**: `LoadByID` fetches all events and replays them via `ApplyEvent()`
- **Immutable events**: Events are append-only; never update or delete from `event_store`
- **Projection = read model**: Query projections for reads, never reconstruct aggregates for read-only operations

## sqlc

- Config: `internal/db/sqlc.yaml`
- Queries go in `internal/db/*.sql` files using the `-- name: QueryName :verb` format
- Schema: `internal/db/zz_generated_schema.sql` (exported from live DB via `mise import-db-schema`)
- Run `mise generate:backend` after editing `.sql` files
- Uses pgx/v5 driver with `google/uuid` UUID type overrides
- Nullable columns generate pointer types

## Database Migrations

- Tool: golang-migrate
- Location: `internal/db/migrations/sql/`
- Files: `<version>_<name>.up.sql` and `<version>_<name>.down.sql`
- Migrations run automatically on startup (controlled by `DATEI_DATABASE_MIGRATIONS` config)
- After adding migrations, run `mise import-db-schema` to update `internal/db/zz_generated_schema.sql` for sqlc
- Run manually: `mise run:datei:migrate`

## Error Handling

- Define sentinel errors in `internal/dateierrors/`
- Wrap errors with context: `fmt.Errorf("failed to do X: %w", err)`
- Check with `errors.Is()`
- Map domain errors to HTTP status codes in endpoint handlers

# Frontend Conventions

## TypeScript Best Practices

- Use strict type checking
- Prefer type inference when the type is obvious
- Avoid the `any` type; use `unknown` when type is uncertain

## Angular Material (Material 3 Expressive)

This project uses Angular Material 21 with Material 3 theming. All UI must follow M3 Expressive conventions.

- Always use Angular Material components (`mat-button`, `mat-card`, `mat-table`, etc.) over custom elements when a Material equivalent exists
- Import Material modules individually per component: `import { MatButtonModule } from '@angular/material/button'`
- Use M3 system-level CSS variables (`--mat-sys-surface`, `--mat-sys-on-surface`, etc.) for colors — do NOT hardcode color values
- Use Material elevation classes (`mat-elevation-z*`) instead of custom `box-shadow`
- Use Material typography classes (`mat-h1`, `mat-body-medium`, etc.) for text styling
- The theme is defined in `frontend/src/material-theme.scss` using `@include mat.theme()` with `mat.$azure-palette` (primary) and `mat.$blue-palette` (tertiary)
- Use `@angular/cdk` utilities (e.g., `BreakpointObserver`) for responsive behavior instead of manual `window.matchMedia`

## Angular Best Practices

- Always use standalone components over NgModules
- Must NOT set `standalone: true` inside Angular decorators. It's the default in Angular v20+.
- Use signals for state management
- Implement lazy loading for feature routes
- Do NOT use the `@HostBinding` and `@HostListener` decorators. Put host bindings inside the `host` object of the `@Component` or `@Directive` decorator instead
- Use `NgOptimizedImage` for all static images.
  - `NgOptimizedImage` does not work for inline base64 images.
- Do NOT use `@angular/animations` (`provideAnimationsAsync`, animation triggers, `[@name]` bindings). Use Angular's built-in animation directives (`animate.enter`, `animate.leave`) instead.

## Accessibility Requirements

- It MUST pass all AXE checks.
- It MUST follow all WCAG AA minimums, including focus management, color contrast, and ARIA attributes.

### Components

- Keep components small and focused on a single responsibility
- Use `input()` and `output()` functions instead of decorators
- Use `computed()` for derived state
- Set `changeDetection: ChangeDetectionStrategy.OnPush` in `@Component` decorator
- Always use external template files (`templateUrl`) — do not use inline `template:`
- Prefer Signal Forms (`@angular/forms/signals`) over Reactive Forms and Template-driven forms (experimental, introduced in Angular 21)
- Do NOT use `ngClass`, use `class` bindings instead
- Do NOT use `ngStyle`, use `style` bindings instead
- When using external templates/styles, use paths relative to the component TS file.

## State Management

- Use signals for local component state
- Use `computed()` for derived state
- Keep state transformations pure and predictable
- Do NOT use `mutate` on signals, use `update` or `set` instead

## Templates

- Keep templates simple and avoid complex logic
- Use native control flow (`@if`, `@for`, `@switch`) instead of `*ngIf`, `*ngFor`, `*ngSwitch`
- Use the async pipe to handle observables
- Do not assume globals like (`new Date()`) are available.
- Do not write arrow functions in templates (they are not supported).

## Services

- Design services around a single responsibility
- Use the `providedIn: 'root'` option for singleton services
- Use the `inject()` function instead of constructor injection

# Maintaining This File

This file is the primary source of truth for how the AI agent works with this codebase. It MUST be kept up to date as the project evolves.

- When adding new packages, directories, or architectural patterns, update the relevant sections above
- When changing the code generation pipeline, build tasks, or migration tooling, reflect those changes here
- When introducing new conventions (error handling, naming, testing patterns), document them
- After any structural refactor, verify that the Project Structure tree and feature checklist still match reality
- If a section becomes inaccurate, fix it immediately — stale instructions cause compounding errors
