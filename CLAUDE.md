# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Goal

Build a Go REST API backed by a local PostgreSQL database that serves Pokemon data from `data/pokemons.json`. The API must support favorites (add/remove/list) and filtering/pagination. A Swagger UI for manual testing is required.

## Technology Stack

- **Language**: Go (idiomatic practices per [Effective Go](https://go.dev/doc/effective_go))
- **Database**: PostgreSQL (local, but designed to be swappable for a cloud SQL provider)
- **API docs**: Swagger/OpenAPI
- **Auth/users**: Single user for now, but schema should leave room to extend to multi-user

## Commands

Once the project is scaffolded, expected commands will be:

```bash
# Run the API server
go run ./cmd/api

# Run all tests
go test ./...

# Run a single test or package
go test ./internal/pokemon/... -run TestGetByID -v

# Seed the database from JSON
go run ./cmd/seed

# Generate Swagger docs (if using swaggo)
swag init -g cmd/api/main.go
```

## Data Structure (`data/pokemons.json`)

151 Generation I Pokemon. Key fields per entry:

| Field | Type | Notes |
|---|---|---|
| `id` | string | Zero-padded, e.g. `"001"` |
| `name` | string | |
| `classification` | string | |
| `types` | string[] | 1–2 types, e.g. `["Grass","Poison"]` |
| `resistant` | string[] | |
| `weaknesses` | string[] | |
| `weight` | `{minimum, maximum}` | String values with unit, e.g. `"6.04kg"` |
| `height` | `{minimum, maximum}` | String values with unit, e.g. `"0.61m"` |
| `fleeRate` | float | |
| `maxCP` | int | |
| `maxHP` | int | |
| `attacks` | `{fast: Attack[], special: Attack[]}` | `Attack` = `{name, type, damage}` |
| `evolutions` | `[{id, name}]` | Optional |
| `Previous evolution(s)` | `[{id, name}]` | Optional, note the unusual key name |
| `evolutionRequirements` | `{amount, name}` | Optional |
| `Pokémon Class` | string | Optional; value is `"This is a LEGENDARY Pokémon."` or `"This is a MYTHIC Pokémon."` — 5 total |
| `Common Capture Area` | string | Optional; 4 Pokemon have this |

Region fields (`Asia`, `North America`, `Western Europe`, `Australia, New Zealand`) appear on exactly 1 Pokemon each and contain the string `"Common Capture Area"` — these are data quirks, not true attributes.

## Recommended Database Schema

Many-to-many relationships (types, resistant, weaknesses, attacks) warrant junction tables rather than arrays/JSONB, to support efficient filtering by type. Suggested normalized tables:

- `pokemons` — core fields (id, name, classification, flee_rate, max_cp, max_hp, pokemon_class)
- `pokemon_types`, `types` — junction + lookup
- `pokemon_resistant`, `pokemon_weaknesses` — junction to type lookup
- `attacks` — fast/special attacks with a category column
- `pokemon_attacks` — junction
- `evolutions` — `(pokemon_id, evolves_to_id)` pairs
- `favorites` — `(pokemon_id)` with a `user_id` FK placeholder for future multi-user support

## API Endpoints Required

| Method | Path | Description |
|---|---|---|
| GET | `/pokemons/:id` | Get by numeric/string id |
| GET | `/pokemons?page=&limit=` | Paginated list |
| GET | `/pokemons?name=` | Search by name |
| GET | `/pokemons?type=&...` | Filter by type and other attributes |
| GET | `/pokemons/types` | List all types |
| POST | `/favorites` | Add a pokemon to favorites |
| DELETE | `/favorites/:id` | Remove from favorites |
| GET | `/favorites` | List all favorites |

## Architecture

Target a layered (or hexagonal) architecture:

```
cmd/
  api/        # main entry point, HTTP server setup
  seed/       # one-time DB seeder from JSON
internal/
  pokemon/    # domain: models, repository interface, service
  favorites/  # domain: models, repository interface, service
  db/         # PostgreSQL connection/migrations
  handler/    # HTTP handlers, request/response DTOs
  swagger/    # generated docs
```

Keep the repository interface abstract so the PostgreSQL implementation can be swapped. Pass `*sql.DB` (or a thin wrapper) via dependency injection rather than global state.
