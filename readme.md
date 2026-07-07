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

2. Create a PostgreSQL database and user matching the connection string in `main.go`:

   ```
   host=localhost user=admin password=admin123 dbname=data-processing-pipeline port=5432 sslmode=disable
   ```

   You can create the database and user with:

   ```sql
   CREATE USER admin WITH PASSWORD 'admin123';
   CREATE DATABASE "data-processing-pipeline" OWNER admin;
   ```

   Tables are created automatically via GORM auto-migration on startup, so no manual migrations are needed.

## Run

Start the server:

```bash
go run main.go
```

The API will be available at `http://localhost:9090`.

## API Endpoints

Base path: `/api/v1/pipelines`

| Method | Path            | Description                             |
| ------ | --------------- | ---------------------------------------- |
| POST   | `/`             | Start a pipeline run (see below)        |
| GET    | `/`             | List pipelines                          |
| GET    | `/:id`          | Get a pipeline                          |
| GET    | `/:id/progress` | Get pipeline progress                   |
| GET    | `/:id/results`  | Get pipeline results                    |
| GET    | `/:id/errors`   | Get pipeline errors                     |
| PATCH  | `/:id/cancel`   | Cancel a running pipeline               |
| PUT    | `/:id`          | Update a pipeline                       |
| DELETE | `/:id`          | Delete a pipeline                       |

### Starting a pipeline run

`POST /api/v1/pipelines` accepts a request describing what to process, and
immediately starts processing in the background (`202 Accepted`, doesn't
block for completion):

```json
{
  "sources": [
    { "type": "csv", "path": "./sample.csv" },
    { "type": "csv", "path": "https://covid.ourworldindata.org/data/owid-covid-data.csv" },
    { "type": "json", "path": "https://jsonplaceholder.typicode.com/posts" },
    { "type": "json", "path": "https://randomuser.me/api/?results=10", "records_path": "results" }
  ],
  "export_type": "json"
}
```

The number of validation/transform workers is fixed in code
(`src/pipelines/types.go`), not configurable per request.

`sources[].type` supports `"csv"` and `"json"`; other types are reported as a
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

The concurrent engine lives in `src/pipelines/` (see
[`src/docs/pipeline-flow.md`](src/docs/pipeline-flow.md) for a walkthrough of
the read -> validate -> transform -> count flow).
