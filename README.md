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

- [Go 1.24+](https://go.dev/dl/)
- [Postgres.app](https://postgresapp.com) (or any PostgreSQL 14+ instance)

## Setup

### 1. Create the database

```bash
createdb pokemon
```

### 2. Seed the database

Reads `data/pokemons.json` and creates all tables and inserts all 151 Pokemon in a single transaction.

```bash
DB_USER=$USER DB_PASSWORD="" go run ./cmd/seed
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
