package db

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

// Config holds PostgreSQL connection parameters.
// Values are read from environment variables in main.go.
type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// Open opens a PostgreSQL connection and verifies it with a ping.
func Open(cfg Config) (*sql.DB, error) {
	// URL format avoids DSN parsing issues with empty password values.
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName, cfg.SSLMode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("db.Ping: %w", err)
	}

	return db, nil
}

// Migrate runs the schema DDL statements required to create all tables.
// It is idempotent — safe to call on every startup.
// Each statement is executed individually because lib/pq does not support
// multiple statements in a single Exec call.
func Migrate(db *sql.DB) error {
	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("migrate: %w\nstatement: %s", err, stmt)
		}
	}
	return nil
}

var statements = []string{
	`CREATE TABLE IF NOT EXISTS types (
		name TEXT PRIMARY KEY
	)`,
	`CREATE TABLE IF NOT EXISTS pokemons (
		id             VARCHAR(3)     PRIMARY KEY,
		name           TEXT           NOT NULL UNIQUE,
		classification TEXT           NOT NULL,
		flee_rate      NUMERIC(5, 4)  NOT NULL,
		max_cp         INTEGER        NOT NULL,
		max_hp         INTEGER        NOT NULL,
		weight_min     TEXT           NOT NULL,
		weight_max     TEXT           NOT NULL,
		height_min     TEXT           NOT NULL,
		height_max     TEXT           NOT NULL,
		pokemon_class  TEXT           NOT NULL DEFAULT ''
	)`,
	`CREATE TABLE IF NOT EXISTS pokemon_types (
		pokemon_id VARCHAR(3) NOT NULL REFERENCES pokemons(id),
		type_name  TEXT       NOT NULL REFERENCES types(name),
		PRIMARY KEY (pokemon_id, type_name)
	)`,
	`CREATE TABLE IF NOT EXISTS pokemon_resistant (
		pokemon_id VARCHAR(3) NOT NULL REFERENCES pokemons(id),
		type_name  TEXT       NOT NULL REFERENCES types(name),
		PRIMARY KEY (pokemon_id, type_name)
	)`,
	`CREATE TABLE IF NOT EXISTS pokemon_weaknesses (
		pokemon_id VARCHAR(3) NOT NULL REFERENCES pokemons(id),
		type_name  TEXT       NOT NULL REFERENCES types(name),
		PRIMARY KEY (pokemon_id, type_name)
	)`,
	`CREATE TABLE IF NOT EXISTS attacks (
		id       SERIAL PRIMARY KEY,
		name     TEXT    NOT NULL,
		type     TEXT    NOT NULL,
		damage   INTEGER NOT NULL,
		category TEXT    NOT NULL CHECK (category IN ('fast', 'special'))
	)`,
	`CREATE TABLE IF NOT EXISTS pokemon_attacks (
		pokemon_id VARCHAR(3) NOT NULL REFERENCES pokemons(id),
		attack_id  INTEGER    NOT NULL REFERENCES attacks(id),
		PRIMARY KEY (pokemon_id, attack_id)
	)`,
	`CREATE TABLE IF NOT EXISTS evolutions (
		pokemon_id      VARCHAR(3) NOT NULL REFERENCES pokemons(id),
		evolves_to_id   INTEGER    NOT NULL,
		evolves_to_name TEXT       NOT NULL,
		direction       TEXT       NOT NULL CHECK (direction IN ('next', 'prev')),
		PRIMARY KEY (pokemon_id, evolves_to_id, direction)
	)`,
	`CREATE TABLE IF NOT EXISTS evolution_requirements (
		pokemon_id VARCHAR(3) PRIMARY KEY REFERENCES pokemons(id),
		amount     INTEGER    NOT NULL,
		item_name  TEXT       NOT NULL
	)`,
	`CREATE TABLE IF NOT EXISTS favorites (
		pokemon_id VARCHAR(3) PRIMARY KEY REFERENCES pokemons(id),
		created_at TIMESTAMP  NOT NULL DEFAULT NOW()
	)`,
}
