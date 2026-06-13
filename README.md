# FinishLine

Backend for a footrace participant registration system. Built with Go, following Hexagonal Architecture with DDD Lite.

## Requirements

- Go 1.26+
- Docker (to run the containerized app)
- [golangci-lint](https://golangci-lint.run/) (linting)

## Getting started

Create a `.env` file in the project root:

```
APP_ENV=development
APP_PORT=8080
DB_HOST=<database host>
DB_USER=<database user>
DB_PASSWORD=<database password>
DB_NAME=<database name>
DB_PORT=5432
DB_SSLMODE=require
```

Then:

```sh
make run          # run locally on :8080
# or
make up           # build and run with Docker
```

In development the schema is auto-migrated at startup.

## API documentation

The API is documented with OpenAPI and served by the app itself:

- **http://localhost:8080/docs** — interactive documentation (browse endpoints, schemas, and test requests right from the browser)
- **http://localhost:8080/openapi.yaml** — raw spec

The spec lives at [api/openapi.yaml](api/openapi.yaml) and is the source of truth for the HTTP contract. Workflow: implement the feature, verify the real responses, then update the spec **in the same PR** — it is hand-maintained, so an outdated spec is a lying spec.

Versioning follows [SemVer](https://semver.org) via `info.version` in the spec.

## Project structure

```
cmd/api/              # entrypoint: wiring (composition root), HTTP server
api/                  # OpenAPI spec, embedded into the binary
internal/
  common/             # cross-cutting: config, postgres, security, server
  <module>/           # one package per feature (user, auth, event, ...)
    domain/           # entities, business rules, domain errors
    ports/            # interfaces (contracts) the module needs
    service/          # use cases, orchestrates domain + ports
    adapters/
      postgres/       # gorm implementation of the repository port
      rest/           # gin handlers + DTOs
  apperr/             # shared domain kernel: categorized errors
```

Rules of the architecture:

- Dependencies always point inward: adapters → service → domain. The domain depends on no framework or infrastructure (no gin, no gorm) — only the standard library, a UUID helper, and the `apperr` kernel.
- Ports are named after roles (`UserRepository`); adapters after technologies (`postgres`, `rest`).
- Adapters translate at the boundary: storage errors → domain errors, domain → DTOs.
- Domain errors carry a category (`apperr.Kind`); the transport layer maps categories to status codes, so handlers never enumerate which errors are 400s.

## Database migrations

In development the schema is applied at startup via `postgres.RunMigrations`, which enables shared extensions (e.g. `citext`) and then runs each module's gorm `AutoMigrate` in order. In production this is skipped — schema changes ship as explicit, versioned migrations.

## Commands

| Command | Description |
|---|---|
| `make run` | Run the API locally |
| `make build` | Build binary to `bin/finishline` |
| `make test` | Run tests |
| `make lint` | Run golangci-lint |
| `make up` | Build image and start with Docker Compose |
| `make down` | Stop Docker Compose |
