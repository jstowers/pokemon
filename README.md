# Pokemon API

A REST API for browsing and favoriting Generation I Pokemon, built with Go and PostgreSQL.

## Technology Choices

**Go** — Chosen for its strong standard library, idiomatic HTTP support (net/http), and the explicit, readable style it encourages. The codebase follows [Effective Go](https://go.dev/doc/effective_go) conventions.

**PostgreSQL** — The Pokemon data has well-defined relationships (types, attacks, evolutions) that map cleanly to a normalized relational schema. Filtering by type, for example, is efficient with a junction table and a simple JOIN, whereas a document store would require scanning arrays. The connection is abstracted behind a `Config` struct so the driver can be swapped for a cloud-managed PostgreSQL instance (AWS RDS, Google Cloud SQL, IBM Cloud databases) by changing environment variables alone.

**Architecture** — Layered (hexagonal) design with three layers:
- **Domain** (`internal/pokemon`) — models, repository interface, and service. No database or HTTP imports.
- **Infrastructure** (`internal/repository`) — PostgreSQL implementation of the repository interface.
- **Delivery** (`internal/handler`) — HTTP handlers, request/response DTOs, and route registration.

The domain defines what it needs via the `Repository` interface. The infrastructure and delivery layers depend on the domain; the domain depends on nothing outside itself. This makes the business logic independently testable without a running database.

## Prerequisites

### 1. Clone the repository

In a folder of your choosing, run the following commands to clone this repo:

```bash
git clone https://github.com/jstowers/pokemon.git
cd pokemon
```

### 2. Install Go

Download and install [Go 1.24+](https://go.dev/dl/).

### 3. Install PostgreSQL

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

**Option B — Homebrew**

If you have Homebrew installed on your machine:

  ```bash
  brew install postgresql@17
  brew services start postgresql@17
  ```

## Build and Run

### 1. Create the database

```bash
createdb pokemon
```

### 2. Seed the database

Reads `data/pokemons.json`, creates all tables, and inserts all 151 Pokemon in a single transaction.

```bash
DB_USER=$USER DB_PASSWORD="" go run ./cmd/seed
```

`DB_USER=$USER` uses your macOS username, which is the default superuser created by both Postgres.app and Homebrew. If your PostgreSQL setup uses a different username or password, set those values explicitly:

```bash
DB_USER=myuser DB_PASSWORD=mypassword go run ./cmd/seed
```

### 3. Start the API

```bash
DB_USER=$USER DB_PASSWORD="" go run ./cmd/api
```

The server starts on port 8080 by default.

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `DB_HOST` | `localhost` | PostgreSQL host |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | `postgres` | PostgreSQL user |
| `DB_PASSWORD` | _(empty)_ | PostgreSQL password |
| `DB_NAME` | `pokemon` | Database name |
| `DB_SSLMODE` | `disable` | SSL mode |
| `ADDR` | `:8080` | HTTP listen address |

## API Endpoints

| Method | Path | Description |
|---|---|---|
| GET | `/pokemons/{id}` | Get a Pokemon by ID (e.g. `001`) |
| GET | `/pokemons` | List Pokemon (paginated) |
| GET | `/pokemons?name=char` | Filter by name (partial, case-insensitive) |
| GET | `/pokemons?type=Fire` | Filter by type |
| GET | `/pokemons?page=2&limit=10` | Pagination controls |
| POST | `/favorites` | Add a Pokemon to favorites |
| DELETE | `/favorites/{id}` | Remove a Pokemon from favorites |
| GET | `/favorites` | List favorite Pokemon (paginated) |

### Example Requests

```bash
# Get Bulbasaur
curl http://localhost:8080/pokemons/001

# Search by name
curl "http://localhost:8080/pokemons?name=char"

# Filter by type
curl "http://localhost:8080/pokemons?type=Fire"

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

To regenerate Swagger docs after changing handler annotations:

```bash
swag init -g cmd/api/main.go --output docs
```

## Running Tests

```bash
# All tests
go test ./...

# A specific package
go test ./internal/pokemon/...

# A single test
go test ./internal/handler/... -run TestAddFavorite_AlreadyFavorite -v
```

Tests cover the service (business logic) and handler (HTTP) layers using a mock repository — no database required to run the test suite.

## Initial Commit

Wednesday, April 29, 2026

## Last Revision

Wednesday, April 29, 2026