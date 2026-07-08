# Data Processing Pipeline

A Go REST API for managing data processing pipelines, built with Gin and GORM.

## Prerequisites

- Go 1.26.4 or later
- PostgreSQL (running locally, or accessible via connection string)

## Setup

1. Clone the repository and install dependencies:

   ```bash
   go mod download
   ```

2. Create a PostgreSQL database and user:

   ```sql
   CREATE USER admin WITH PASSWORD 'admin123',
   CREATE DATABASE "data-processing-pipeline" OWNER admin,
   ```

   Tables are created automatically via GORM auto-migration on startup, so no manual migrations are needed.

3. Copy `.env.example` to `.env` and adjust the values if your database or
   server address differ from the defaults:

   ```bash
   cp .env.example .env
   ```

   | Variable          | Default                    | Description                                                                                                                                                                                     |
   | ----------------- | -------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
   | `SERVER_ADDR`     | `localhost:9090`           | Address the HTTP server listens on                                                                                                                                                              |
   | `DB_HOST`         | `localhost`                | Postgres host                                                                                                                                                                                   |
   | `DB_USER`         | `admin`                    | Postgres user                                                                                                                                                                                   |
   | `DB_PASSWORD`     | `admin123`                 | Postgres password                                                                                                                                                                               |
   | `DB_NAME`         | `data-processing-pipeline` | Postgres database name                                                                                                                                                                          |
   | `DB_PORT`         | `5432`                     | Postgres port                                                                                                                                                                                   |
   | `DB_SSLMODE`      | `disable`                  | Postgres `sslmode`                                                                                                                                                                              |
   | `WORKER_ADDR`     | `localhost:9091`           | Address the worker's internal API listens on                                                                                                                                                    |
   | `WORKER_URL`      | `http://localhost:9091`    | URL the server uses to reach the worker's internal API                                                                                                                                          |
   | `EXPORT_BASE_URL` | `http://localhost:9091`    | Public URL used to build `export_url` links returned to API clients, must be reachable from wherever clients call the API (unlike `WORKER_URL`, which may point at an internal Docker hostname) |

   `.env` is gitignored — never commit real credentials.

## Run

Start the server:

```bash
go run ./apps/server/cmd
```

The API will be available at `http://localhost:9090`.

Start the worker (in a separate terminal):

```bash
go run ./apps/worker/cmd
```

## Development

This project includes an `.air.toml` config for [Air](https://github.com/air-verse/air),
which watches `.go` files and automatically rebuilds and restarts the server
on save (hot reload), instead of manually re-running `go run ./apps/server/cmd`.

1. Install Air (once):

   ```bash
   go install github.com/air-verse/air@latest
   ```

   Make sure your Go bin directory (`go env GOPATH`/bin) is on your `PATH` so
   the `air` command is available.

2. Start the dev server with hot reload:

   ```bash
   air
   ```

   Air uses the config in `.air.toml`: it builds to `./tmp/main` (`.exe` on
   Windows), excludes `tmp/`, `vendor/`, `testdata/` and `exports/` from
   watching, and gracefully restarts the server on each change.

This step is optional — `go run ./apps/server/cmd` still works for a one-off run.

## Docker

The project includes `apps/server/Dockerfile`, `apps/worker/Dockerfile`, and a
`docker-compose.yml` that build the API and worker and run them alongside a
Postgres container — no local Go or Postgres install required.

1. Build and start all services:

   ```bash
   docker compose up --build
   ```

   This starts:
   - `db`: `postgres:16-alpine`, seeded with the same credentials as
     `.env.example` (`admin` / `admin123` / `data-processing-pipeline`)
   - `app`: the API, built from `apps/server/Dockerfile`, listening on
     `http://localhost:9090`
   - `worker`: the pipeline worker, built from `apps/worker/Dockerfile`,
     exposing an internal API on `http://worker:9091` that `app` uses to
     enqueue jobs and read live progress/cancel state (kept in the worker's
     own memory — no external queue or cache)

   Database tables are created automatically via GORM auto-migration on
   startup, same as running locally.

2. Generated exports are written to `./exports` on the host (mounted into the
   container at `/app/exports`), so pipeline results persist across
   `docker compose up`/`down` cycles. Postgres data persists in the
   `db_data` named volume.

3. To run only the API image against a database you manage yourself, build
   and run it directly, overriding env vars as needed:

   ```bash
   docker build -t data-processing-pipeline -f apps/server/Dockerfile .
   docker run --rm -p 9090:9090 \
     -e SERVER_ADDR=0.0.0.0:9090 \
     -e DB_HOST=host.docker.internal \
     -e DB_USER=admin \
     -e DB_PASSWORD=admin123 \
     -e DB_NAME=data-processing-pipeline \
     -e DB_PORT=5432 \
     -e DB_SSLMODE=disable \
     data-processing-pipeline
   ```

   Note `SERVER_ADDR` must bind to `0.0.0.0`, not `localhost`, for the
   container's published port to be reachable from the host.

4. Redeploying after a code change: containers already running via
   `docker compose up -d` keep using the image they were created from — a
   plain `docker compose restart` does **not** recompile the Go binary. Rebuild
   the image first, then recreate the container:

   ```bash
   docker compose build app
   docker compose up -d app
   ```

   (Swap `app` for `worker` if the change is in `apps/worker/`.) Compose
   detects the new image tag and recreates only that service, leaving `db`
   untouched.

## API Documentation

A Swagger UI is served directly by the running API at `/docs` (no `/api/v1`
prefix, no API key required), rendered from `docs/swagger-ui.html`.

Once the server is running, open `http://localhost:9090/docs` in a browser
to explore and try the API interactively.

## API Endpoints

Base path: `/api/v1/pipelines`

| Method | Path            | Description                      |
| ------ | --------------- | -------------------------------- |
| POST   | `/`             | Start a pipeline run (see below) |
| GET    | `/`             | List pipelines                   |
| GET    | `/:id`          | Get a pipeline                   |
| GET    | `/:id/progress` | Get pipeline progress            |
| GET    | `/:id/results`  | Get pipeline results             |
| GET    | `/:id/errors`   | Get pipeline errors              |
| PATCH  | `/:id/cancel`   | Cancel a running pipeline        |
| PUT    | `/:id`          | Update a pipeline                |
| DELETE | `/:id`          | Delete a pipeline                |

### Starting a pipeline run

`POST /api/v1/pipelines` accepts a request describing what to process, and
immediately starts processing in the background (`202 Accepted`, doesn't
block for completion):

```json
{
  "sources": [
    { "type": "csv", "path": "./sample.csv" },
    {
      "type": "csv",
      "path": "https://covid.ourworldindata.org/data/owid-covid-data.csv"
    },
    { "type": "json", "path": "https://jsonplaceholder.typicode.com/posts" },
    {
      "type": "json",
      "path": "https://randomuser.me/api/?results=10",
      "records_path": "results"
    }
  ],
  "export_type": "json"
}
```

The number of validation/transform workers is fixed in code
(`apps/worker/types.go`), not configurable per request.

`sources[].type` supports `"csv"` and `"json"`, other types are reported as a
pipeline error and skipped. `path` can be a local file path or an `http(s)://`
URL for either type. For a `"json"` source:

- a plain JSON array response (e.g. JSONPlaceholder, CoinGecko) yields one
  record per element
- a single JSON object response (e.g. Open-Meteo) yields exactly one record
- an object that wraps the list under a key (e.g. RandomUser's
  `{"results": [...]}`) needs `records_path` set to that key

All sources across the request are read concurrently and merged onto the same
channel, so mixing CSV and JSON sources in one pipeline run (e.g. to build a
combined report from several data sets) works out of the box. Progress and
results are available via the existing `GET /:id/progress`, `GET /:id/results`
and `GET /:id/errors` endpoints, and the final count is also written to
`exports/<pipeline-id>.json`.

The concurrent engine lives in `apps/worker/` (see
[`src/docs/pipeline-flow.md`](src/docs/pipeline-flow.md) for a walkthrough of
the read -> validate -> transform -> count flow).
