# Pokemon API

A REST API for browsing Generation I Pokemon and adding favorites, built with Go and PostgreSQL.

Joseph Stowers

Monday, May 4, 2026

## Table of Contents

[Technology Choices](#technology-choices)

[Prerequisites](#prerequisites)

[Build and Run](#build-and-run)

[API Endpoints](#api-endpoints)

[Swagger UI](#swagger-ui)

[Unit and Integration Tests](#unit-and-integration-tests)

[Reference](#reference)

[Initial Commit](#initial-commit)

[Last Revision](#last-revision)

## Technology Choices

**Go** — Chosen for its strong standard library, idiomatic HTTP support (net/http), and the explicit, readable style it encourages. The codebase follows [Effective Go](https://go.dev/doc/effective_go) conventions.

**PostgreSQL** — The Pokemon data has well-defined relationships (types, attacks, evolutions) that map cleanly to a normalized relational schema. Filtering by type, for example, is efficient with a junction table and a simple JOIN, whereas a document store would require scanning arrays. The connection is abstracted behind a `Config` struct so the driver can be swapped for a cloud-managed PostgreSQL instance (AWS RDS, Google Cloud SQL, IBM Cloud databases) by changing environment variables alone.

**Architecture** — Layered (hexagonal) design with three layers:
- **Domain** (`internal/pokemon`) — models, repository interface, and service. No database or HTTP imports.
- **Infrastructure** (`internal/repository`) — PostgreSQL implementation of the repository interface.
- **Delivery** (`internal/handler`) — HTTP handlers, request/response DTOs, and route registration.

The domain defines what it needs via the `Repository` interface. The infrastructure and delivery layers depend on the domain; the domain depends on nothing outside itself. This makes the business logic independently testable without a running database.

## Prerequisites

### 1. Install Go

Download and install [Go 1.24.4](https://go.dev/dl/) or newer.

### 2. Install PostgreSQL

You need a local PostgreSQL database to test the API.  You have two installation options:

**Option A — Postgres.app (macOS, recommended)**

1. Download and install [Postgres.app](https://postgresapp.com). Open the app, click **Initialize**, then **Start**.

2. Add the CLI tools to your PATH so that `createdb` and `psql` are available in the terminal:

    ```bash
    sudo mkdir -p /etc/paths.d
    ```

    ```bash
    echo /Applications/Postgres.app/Contents/Versions/latest/bin | sudo tee /etc/paths.d/postgresapp
    ```

3. Open a new terminal window for the PATH change to take effect.

4. Check for a successful installation of these tools with the following commands:

    ```bash
    # show version
    createdb --version
    ```

    ```bash
    # show executable PATH
    which createdb
    ```

    ```bash
    # show version
    psql --version
    ```

    ```bash
    # show executable PATH
    which psql
    ``` 

**Option B — Homebrew**

If you have [Homebrew](https://brew.sh) installed on your machine:

  ```bash
  brew install postgresql@17
  brew services start postgresql@17
  ```

### 3. Clone the repository

In a folder of your choosing, run the following command to clone this repo:

```bash
git clone https://github.com/jstowers/pokemon.git
```

```
cd pokemon
```

## Build and Run

### 1. Create the database

```bash
createdb pokemon
```

### 2. Configure the environment

The application reads its configuration from a `.env.dev` file in the project root:

```text
DB_HOST=localhost
DB_PORT=5432
DB_USER=
DB_PASSWORD=
DB_NAME=pokemon
DB_SSLMODE=disable
ADDR=:8080
```

Copy the example file `.env.example` to `.env.dev`:

```bash
cp .env.example .env.dev
```

Open `.env.dev` and set your PostgreSQL credentials for `DB_USER` and `DB_PASSWORD`. With Postgres.app or Homebrew, the default `DB_USER` is your macOS username and `DB_PASSWORD` can be left empty.

### 3. Seed the database

Reads `data/pokemons.json`, creates all tables, and inserts all 151 Pokemon in a single transaction:

```bash
go run ./cmd/seed
```

### 4. Start the API server

```bash
go run ./cmd/api
```

### 5. Open Swagger API

In a browser, open the following URL: http://localhost:8080/swagger/index.html

![swagger-pokemon-api](/image/swagger-pokemon-api.png)

## API Endpoints

| Method | Path | Description |
|---|---|---|
| GET | `/pokemons/{id}` | Get a Pokemon by ID (e.g. `001`) |
| GET | `/pokemons` | List Pokemon (paginated) |
| GET | `/pokemons?name=char` | Filter by name (partial, case-insensitive) |
| GET | `/pokemons?type=Fire` | Filter by type |
| GET | `/pokemons?heightMin=0.5&heightMax=2.0` | Filter by height range (m) |
| GET | `/pokemons?page=2&limit=10` | Pagination controls |
| POST | `/favorites` | Add a Pokemon to favorites |
| DELETE | `/favorites/{id}` | Remove a Pokemon from favorites |
| GET | `/favorites` | List favorite Pokemon (paginated) |

### Filter Parameters

The `/pokemons` endpoint supports flexible filtering with optional, combinable query parameters.

| Parameter | Type | Description |
|---|---|---|
| `name` | string | Partial, case-insensitive name match |
| `type` | string | Exact type match (e.g. `Fire`, `Water`, `Grass`) |
| `weakness` | string | Exact weakness match |
| `resistant` | string | Exact resistance match |
| `fleeRateMin` | float | Minimum flee rate (0–1), inclusive |
| `fleeRateMax` | float | Maximum flee rate (0–1), inclusive |
| `weightMin` | float | Minimum weight in kg (compared against the Pokemon's minimum weight) |
| `weightMax` | float | Maximum weight in kg (compared against the Pokemon's maximum weight) |
| `heightMin` | float | Minimum height in m (compared against the Pokemon's minimum height) |
| `heightMax` | float | Maximum height in m (compared against the Pokemon's maximum height) |
| `page` | int | Page number (default `1`) |
| `limit` | int | Results per page, max 100 (default `20`) |

### Example Filter Requests

| Method | Path | Description |
|---|---|---|
| GET | `/pokemons?type=Fire` | Filter by type |
| GET | `/pokemons?weakness=Fire` | Filter by weakness |
| GET | `/pokemons?resistant=Water` | Filter by resistance |
| GET | `/pokemons?fleeRateMin=0.1&fleeRateMax=0.5` | Filter by flee rate range |
| GET | `/pokemons?weightMin=5&weightMax=50` | Filter by weight range (kg) |

### Example Requests

```bash
# Get Bulbasaur (id 001)
curl http://localhost:8080/pokemons/001

# Search by name "char" — returns Charmander, Charmeleon, Charizard
curl "http://localhost:8080/pokemons?name=char"

# Filter by type "Fire"
curl "http://localhost:8080/pokemons?type=Fire"

# Fire-type Pokemon that are also weak to Water
curl "http://localhost:8080/pokemons?type=Fire&weakness=Water"

# Lightweight, fast-fleeing Pokemon — under 5 kg and flee rate above 0.15
curl "http://localhost:8080/pokemons?weightMax=5&fleeRateMin=0.15"

# Small Grass-type Pokemon resistant to Water — height under 1 m
curl "http://localhost:8080/pokemons?type=Grass&resistant=Water&heightMax=1.0"

# Add a favorite
curl -X POST http://localhost:8080/favorites \
  -H "Content-Type: application/json" \
  -d '{"pokemonId": "006"}'

# List favorites
curl http://localhost:8080/favorites

# Remove a favorite
curl -X DELETE http://localhost:8080/favorites/006
```

## Swagger UI

Interactive API documentation is available at:

```
http://localhost:8080/swagger/index.html
```

To regenerate Swagger docs after changing handler annotations, first install the `swag` CLI if you haven't already:

```bash
go install github.com/swaggo/swag/cmd/swag@latest
```

Add the Go bin directory to your PATH (add this line to `~/.zshrc` and then run `source ~/.zshrc`):

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

Then regenerate the docs:

```bash
swag init -g cmd/api/main.go --output docs
```

## Unit and Integration Tests

### Unit tests (no database required)

```bash
# All unit tests
go test ./...

# A specific package
go test ./internal/pokemon/...

# A single test
go test ./internal/handler/... -run TestAddFavorite_AlreadyFavorite -v
```

Unit tests cover the service (business logic) and handler (HTTP) layers using a mock repository.

### Integration Tests

The test suite seeds a small set of fixture Pokemon into a local `pokemon_test` database.

The suite runs all tests and then truncates the fixture data, so each run starts from a clean state.

1. Create the local test database once:

    ```bash
    createdb pokemon_test
    ```

1. Run the integration tests:

    ```bash
    go test ./internal/repository/... -tags integration
    ```

By default, the tests connect to `localhost:5432` with your macOS username and no password (matching Postgres.app defaults). 

Override any value with `TEST_DB_*` environment variables:

  ```bash
  TEST_DB_USER=myuser TEST_DB_PASSWORD=secret go test ./internal/repository/... -tags integration
  ```

## Reference

The [`specification/`](/specification/) folder includes the following:

1. Original `PROBLEM-STATEMENT`.

1. My original prompt `SPECIFICATION` to Claude.

1. The Claude-generated workplan `CLAUDE`.

1. My `WORKLOG` of tasks completed and time spent.

## Initial Commit

Wednesday, April 29, 2026

## Last Revision

Monday, May 4, 2026