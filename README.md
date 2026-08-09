# gorest

HTTP API on [huma](https://github.com/danielgtaylor/huma) v2 (typed handlers, auto OpenAPI) + Postgres, with one resource (`users`) wired end to end as a template for adding more.

## Requirements

- Go 1.26.2 ([go.dev/doc/install](https://go.dev/doc/install))
- Postgres 18 (or compatible)

## Setup

```bash
cp example.env .env
```

Start Postgres + app via the included `docker-compose.yml` (db `app`, user/pass `postgres`):

```bash
docker compose up -d
```

Set `POSTGRES_MIGRATE=true` in `.env` to run `migrations/*.up.sql` automatically on startup (otherwise apply them yourself with [golang-migrate](https://github.com/golang-migrate/migrate)).

## Run

```bash
go run ./cmd
```

Server listens on `HTTP_HOST:HTTP_PORT` (default `localhost:8080`):

| | |
|---|---|
| Docs UI | `http://localhost:8080/docs` |
| Swagger UI | `http://localhost:8080/swagger` |
| OpenAPI spec | `http://localhost:8080/openapi.json` |

```bash
curl -X POST localhost:8080/users \
  -H 'Content-Type: application/json' \
  -d '{"username":"homer","email":"homer@example.com","firstName":"Homer","lastName":"Simpson"}'

curl localhost:8080/users
```

## Configuration

Every setting is an env var (or `.env` key — see `example.env` for the full list with defaults):

| Var | Default | |
|---|---|---|
| `APP_VERSION` | `dev` | reported in every response's `meta.version` |
| `HTTP_HOST` / `HTTP_PORT` | `localhost` / `8080` | |
| `LOG_LEVEL` | `info` | `debug`/`info`/`warn`/`error` |
| `CORS_ALLOWED_ORIGINS` | `*` | comma-separated, or `*` for any origin |
| `POSTGRES_HOST` / `_PORT` / `_USER` / `_PASSWORD` / `_DB` / `_SSLMODE` | localhost defaults | |
| `POSTGRES_MIGRATE` | `false` | run pending migrations on startup |
| `TRACING_ENABLED` | `false` | master switch for OTLP trace/metric/log export |
| `SHUTDOWN_TIMEOUT` | `5s` | grace period for in-flight requests on SIGINT/SIGTERM |

## Development

```bash
go build ./...   # compile
go mod tidy       # sync go.mod/go.sum after changing imports
```

Db table bindings (`pkg/db/model`) are generated via [bob](https://bobg.dev):

```bash
go install github.com/stephenafamo/bob/gen/bobgen-sql@v0.49.0

bobgen-sql -c bobgen.yaml
```

No tests currently exist in this repo.

## Architecture

Layered Go HTTP API on [huma](https://github.com/danielgtaylor/huma) v2 (typed request/response, auto OpenAPI) over the stdlib `net/http.ServeMux` router (via `huma/v2/adapters/humago`), Postgres via [bob](https://bobg.dev)/pgx, and OpenTelemetry tracing/metrics/logs. One resource (`users`) is wired end to end as the template for adding more.

Request flow: `internal/server` (huma handler) → `internal/service/<resource>` (business rules) → `internal/repository/<resource>` (bob queries) → Postgres. Each layer has its own error vocabulary; see "Errors" below for how they translate across boundaries.

- **`cmd/main.go`** — entrypoint. Builds a `signal.NotifyContext` (SIGINT/SIGTERM) and calls `server.Run(ctx)`.
- **`config/config.go`** + **`pkg/config`** — `config.Load()` reads env vars (via `godotenv.Load()` first, so a `.env` file works transparently) into `config.Config` (`pkgconfig.App` plus server-only fields like `ShutdownTimeout`, `AllowedOrigins`). See `example.env` for every key and its default.
- **`internal/server/serve.go`** — `Run()` wires every dependency (tracing → logger → postgres → router → middleware) and blocks in `httpSrv.ListenAndServe()`. This is the one place that knows the full dependency graph.
- **`internal/server/server.go`** — `Server` struct holds per-resource services (e.g. `userSvc`); `NewServer(...)` constructs it. Add a new field here per resource.
- **`internal/server/<resource>_controller.go`** — huma operations for one resource: `registerXRoutes(api, basePath)` calls `huma.Register` per operation, then the handler methods. Handler signature is `func(context.Context, *Input) (*Output, error)` — huma reflects on the Input/Output struct tags (`path`, `query`, `json`, ...) to generate validation + OpenAPI, so a new endpoint is a new Input/Output pair (in `internal/dto`) plus one `huma.Register` call, never manual param parsing.
- **`internal/dto`** — huma Input structs and any resource-specific data shapes (e.g. `UserDTO`). Response bodies use the common `pkg/dto/response.Output[T]`/`response.NewError`, not per-operation Output structs — see "Responses & errors" below.
- **`internal/service/<resource>`** — business rules; takes plain scalar params (not dto structs) so it stays usable from a future non-HTTP transport too. Translates repo errors → `pkg/error/serviceerr`.
- **`internal/repository/<resource>`** — bob queries against `pkg/db/model` (generated table bindings). Speaks the DB's native model types only, never dto types. Translates driver errors (e.g. Postgres unique-violation) → package-local sentinel errors; returns `pkg/error/repoerr` for generic cases (not-found).
- **`internal/converter`** — model ↔ dto translation, used by the service layer.

### Responses & errors

- **`pkg/dto/response`** — the common response envelope, transport-agnostic (not tied to huma).
  - `Output[T]{Body: Response[T]{Meta, Data}}` — every success handler returns `*Output[T]`, built via `response.NewOutput(ctx, data)`. `Meta` (RequestID, TraceID/SpanID, Version, RequestAt/ResponseAt) is read back out of context, stamped there per-request by `middleware.Metadata`.
  - `response.NewError(ctx, err)` — call as `return nil, response.NewError(ctx, err)` from every handler. It logs 5xx/unclassified errors at Error level (via `pkg/log.FromContext(ctx)`, itself stamped into context by `middleware.Logger`) and converts the error into `*ErrorOutput` (huma's own `*huma.ErrorModel`, embedded, plus a top-level `Meta` field) — necessary because huma writes a handler's returned error directly as the JSON body when it implements `StatusError`, and `serviceerr.Error`'s fields are deliberately unexported (would otherwise serialize as `{}`).
  - `pkg/dto/request.Pagination` — embed in a list operation's Input for `page`/`limit` query params; `pkg/dto/response.PaginatedData[T]` is the matching data shape (`Items`, `Total`, `Page`, `Limit`, `TotalPages`).
- **`pkg/error/serviceerr`** — `*Error` wraps one of the base sentinel errors (`ErrNotFound`, `ErrConflict`, `ErrInvalidArgument`, ...) plus a message and optional field-level `Details()` (`AddDetail(field, code, message)`). Implements huma's `StatusError` (`GetStatus()`, derived from the wrapped base error) and gRPC's `GRPCStatus()`, so one error type works for both transports. Construct via `serviceerr.NewNotFound(err)`, `.NewConflict(err)`, etc., not `NewError` directly.
- **`pkg/error/repoerr`** — generic, storage-agnostic sentinels only (`ErrNotFound`, `ErrExisted`). A repo needing something more specific (e.g. "username already taken", detected via a named unique constraint) defines that sentinel in its own package instead — see `internal/repository/user` for the pattern (Postgres unique-violation → `pgerrcode.UniqueViolation` + `pgErr.ConstraintName` → `ErrUsernameExisted`/`ErrEmailExisted`).

### Middleware (`pkg/middleware`)

Applied in `serve.go` as `CORS(...)( Metadata(...)( Logger(logger)(router) ) )`:
- `Metadata(version)` — stamps a fresh `response.Meta` into context per request.
- `Logger(logger)` — stamps `logger` into context (`pkg/log.WithLogger`) and logs one line per request (method, path, status, duration, requestId); Error level on 5xx.
- `CORS(allowedOrigins)` — hand-rolled (`["*"]` = any origin), answers `OPTIONS` preflight directly.

### Logging (`pkg/log`)

`NewLogger(cfg, extra...)` fans out to console (colorized `tint` text handler on a real terminal, JSON otherwise — see `pkg/log/logger.go`) plus any extra `slog.Handler`s (e.g. `tracing.Service.Handler()` for OTEL log export; nil-safe). `Bootstrap()` is a bare stderr logger for use before config/tracing have initialized. `WithLogger`/`FromContext` thread a logger through `context.Context`.

### Observability (`pkg/tracing`)

`tracing.NewService(ctx, &cfg.App)` sets up OTLP gRPC exporters for trace/metric/log per `cfg.Tracing` (master `Enabled` switch, then per-signal `Trace`/`Metric`/`Log`). No-op (zero-value `Service`) when disabled. `WithSkipTrace(ctx)` opts a call out of tracing.

### Database (`pkg/postgres`, `pkg/db/model`)

`postgres.New(ctx, &cfg.App, logger)` connects via pgxpool and, if `cfg.Postgres.IsMigrateSchema`, runs `Migrate()` (golang-migrate against `migrations/*.up.sql`) before returning, logging what ran. `pkg/db/model` is bob-generated table bindings (do not hand-edit; regenerate instead) — `pkg/db/dberror`/`pkg/db/dbinfo` are bob-generated companions.

### Not yet wired into `Run`

`pkg/cache/redis`, `pkg/cache/redsync`, `pkg/cron` exist but aren't constructed in `serve.go` today — check before assuming they're active.

## Adding a new resource

1. `internal/dto`: Input structs + any resource-specific data shape.
2. `internal/repository/<resource>`: bob queries against `pkg/db/model`; map driver-specific errors to local/`repoerr` sentinels.
3. `internal/service/<resource>`: business rules; map repo errors to `serviceerr`.
4. `internal/server/<resource>_controller.go`: `registerXRoutes` + handlers, each returning `response.NewOutput(ctx, ...)` / `response.NewError(ctx, err)`.
5. Add the service to `Server` (`server.go`) and call `registerXRoutes` in `serve.go`.
