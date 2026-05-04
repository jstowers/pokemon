package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jstowers/pokemon/internal/pokemon"
)

// PostgresPokemonRepository implements pokemon.Repository using PostgreSQL.
type PostgresPokemonRepository struct {
	db *sql.DB
}

// New constructs a PostgresPokemonRepository.
func New(db *sql.DB) *PostgresPokemonRepository {
	return &PostgresPokemonRepository{db: db}
}

// GetByID fetches a single Pokemon by its string ID (e.g. "001").
func (r *PostgresPokemonRepository) GetByID(ctx context.Context, id string) (*pokemon.Pokemon, error) {
	const q = `
		SELECT id, name, classification, flee_rate, max_cp, max_hp,
		       weight_min, weight_max, height_min, height_max, pokemon_class
		FROM pokemons WHERE id = $1`

	row := r.db.QueryRowContext(ctx, q, id)
	p, err := scanPokemon(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, pokemon.ErrNotFound
		}
		return nil, fmt.Errorf("GetByID: %w", err)
	}

	if err := r.populate(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// GetByName fetches a single Pokemon by name (case-insensitive).
func (r *PostgresPokemonRepository) GetByName(ctx context.Context, name string) (*pokemon.Pokemon, error) {
	const q = `
		SELECT id, name, classification, flee_rate, max_cp, max_hp,
		       weight_min, weight_max, height_min, height_max, pokemon_class
		FROM pokemons WHERE LOWER(name) = LOWER($1)`

	row := r.db.QueryRowContext(ctx, q, name)
	p, err := scanPokemon(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, pokemon.ErrNotFound
		}
		return nil, fmt.Errorf("GetByName: %w", err)
	}

	if err := r.populate(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// List returns a filtered, paginated slice of Pokemon.
func (r *PostgresPokemonRepository) List(ctx context.Context, filter pokemon.Filter, page pokemon.Page) ([]*pokemon.Pokemon, error) {
	q, args := buildListQuery(filter, page)
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("List query: %w", err)
	}
	defer rows.Close()

	return r.collectRows(ctx, rows)
}

// Count returns the total number of Pokemon matching the filter.
func (r *PostgresPokemonRepository) Count(ctx context.Context, filter pokemon.Filter) (int, error) {
	q, args := buildCountQuery(filter)
	var total int
	if err := r.db.QueryRowContext(ctx, q, args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("Count: %w", err)
	}
	return total, nil
}

// AddFavorite inserts a pokemon_id into the favorites table.
func (r *PostgresPokemonRepository) AddFavorite(ctx context.Context, pokemonID string) error {
	const q = `INSERT INTO favorites (pokemon_id) VALUES ($1)`
	if _, err := r.db.ExecContext(ctx, q, pokemonID); err != nil {
		return fmt.Errorf("AddFavorite: %w", err)
	}
	return nil
}

// RemoveFavorite deletes a row from the favorites table.
func (r *PostgresPokemonRepository) RemoveFavorite(ctx context.Context, pokemonID string) error {
	const q = `DELETE FROM favorites WHERE pokemon_id = $1`
	if _, err := r.db.ExecContext(ctx, q, pokemonID); err != nil {
		return fmt.Errorf("RemoveFavorite: %w", err)
	}
	return nil
}

// IsFavorite returns true if the Pokemon is in the favorites table.
func (r *PostgresPokemonRepository) IsFavorite(ctx context.Context, pokemonID string) (bool, error) {
	const q = `SELECT EXISTS (SELECT 1 FROM favorites WHERE pokemon_id = $1)`
	var exists bool
	if err := r.db.QueryRowContext(ctx, q, pokemonID).Scan(&exists); err != nil {
		return false, fmt.Errorf("IsFavorite: %w", err)
	}
	return exists, nil
}

// ListFavorites returns a paginated list of favorited Pokemon.
func (r *PostgresPokemonRepository) ListFavorites(ctx context.Context, page pokemon.Page) ([]*pokemon.Pokemon, error) {
	const q = `
		SELECT p.id, p.name, p.classification, p.flee_rate, p.max_cp, p.max_hp,
		       p.weight_min, p.weight_max, p.height_min, p.height_max, p.pokemon_class
		FROM pokemons p
		INNER JOIN favorites f ON f.pokemon_id = p.id
		ORDER BY f.created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.db.QueryContext(ctx, q, page.Limit, page.Offset)
	if err != nil {
		return nil, fmt.Errorf("ListFavorites query: %w", err)
	}
	defer rows.Close()

	return r.collectRows(ctx, rows)
}

// CountFavorites returns the total number of favorited Pokemon.
func (r *PostgresPokemonRepository) CountFavorites(ctx context.Context) (int, error) {
	const q = `SELECT COUNT(*) FROM favorites`
	var total int
	if err := r.db.QueryRowContext(ctx, q).Scan(&total); err != nil {
		return 0, fmt.Errorf("CountFavorites: %w", err)
	}
	return total, nil
}

// ---- helpers ----------------------------------------------------------------

func scanPokemon(row *sql.Row) (*pokemon.Pokemon, error) {
	p := &pokemon.Pokemon{}
	return p, row.Scan(
		&p.ID, &p.Name, &p.Classification, &p.FleeRate,
		&p.MaxCP, &p.MaxHP,
		&p.Weight.Minimum, &p.Weight.Maximum,
		&p.Height.Minimum, &p.Height.Maximum,
		&p.PokemonClass,
	)
}

func scanPokemonRow(rows *sql.Rows) (*pokemon.Pokemon, error) {
	p := &pokemon.Pokemon{}
	return p, rows.Scan(
		&p.ID, &p.Name, &p.Classification, &p.FleeRate,
		&p.MaxCP, &p.MaxHP,
		&p.Weight.Minimum, &p.Weight.Maximum,
		&p.Height.Minimum, &p.Height.Maximum,
		&p.PokemonClass,
	)
}

// populate loads all related data (types, attacks, evolutions) for a Pokemon.
func (r *PostgresPokemonRepository) populate(ctx context.Context, p *pokemon.Pokemon) error {
	var err error

	p.Types, err = r.fetchStringSlice(ctx, `SELECT type_name FROM pokemon_types WHERE pokemon_id = $1`, p.ID)
	if err != nil {
		return err
	}
	p.Resistant, err = r.fetchStringSlice(ctx, `SELECT type_name FROM pokemon_resistant WHERE pokemon_id = $1`, p.ID)
	if err != nil {
		return err
	}
	p.Weaknesses, err = r.fetchStringSlice(ctx, `SELECT type_name FROM pokemon_weaknesses WHERE pokemon_id = $1`, p.ID)
	if err != nil {
		return err
	}
	p.Attacks.Fast, err = r.fetchAttacks(ctx, p.ID, "fast")
	if err != nil {
		return err
	}
	p.Attacks.Special, err = r.fetchAttacks(ctx, p.ID, "special")
	if err != nil {
		return err
	}
	p.Evolutions, err = r.fetchEvolutions(ctx, p.ID, "next")
	if err != nil {
		return err
	}
	p.PrevEvolutions, err = r.fetchEvolutions(ctx, p.ID, "prev")
	if err != nil {
		return err
	}
	p.EvolutionReq, err = r.fetchEvolutionReq(ctx, p.ID)
	return err
}

func (r *PostgresPokemonRepository) fetchStringSlice(ctx context.Context, q, pokemonID string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, q, pokemonID)
	if err != nil {
		return nil, fmt.Errorf("fetchStringSlice: %w", err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *PostgresPokemonRepository) fetchAttacks(ctx context.Context, pokemonID, category string) ([]pokemon.Attack, error) {
	const q = `
		SELECT a.name, a.type, a.damage
		FROM attacks a
		INNER JOIN pokemon_attacks pa ON pa.attack_id = a.id
		WHERE pa.pokemon_id = $1 AND a.category = $2`

	rows, err := r.db.QueryContext(ctx, q, pokemonID, category)
	if err != nil {
		return nil, fmt.Errorf("fetchAttacks: %w", err)
	}
	defer rows.Close()

	var out []pokemon.Attack
	for rows.Next() {
		var a pokemon.Attack
		if err := rows.Scan(&a.Name, &a.Type, &a.Damage); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *PostgresPokemonRepository) fetchEvolutions(ctx context.Context, pokemonID, direction string) ([]pokemon.Evolution, error) {
	const q = `SELECT evolves_to_id, evolves_to_name FROM evolutions WHERE pokemon_id = $1 AND direction = $2`

	rows, err := r.db.QueryContext(ctx, q, pokemonID, direction)
	if err != nil {
		return nil, fmt.Errorf("fetchEvolutions: %w", err)
	}
	defer rows.Close()

	var out []pokemon.Evolution
	for rows.Next() {
		var e pokemon.Evolution
		if err := rows.Scan(&e.ID, &e.Name); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *PostgresPokemonRepository) fetchEvolutionReq(ctx context.Context, pokemonID string) (*pokemon.EvolutionRequirement, error) {
	const q = `SELECT amount, item_name FROM evolution_requirements WHERE pokemon_id = $1`
	var req pokemon.EvolutionRequirement
	err := r.db.QueryRowContext(ctx, q, pokemonID).Scan(&req.Amount, &req.Name)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("fetchEvolutionReq: %w", err)
	}
	return &req, nil
}

func (r *PostgresPokemonRepository) collectRows(ctx context.Context, rows *sql.Rows) ([]*pokemon.Pokemon, error) {
	var pokemons []*pokemon.Pokemon
	for rows.Next() {
		p, err := scanPokemonRow(rows)
		if err != nil {
			return nil, err
		}
		if err := r.populate(ctx, p); err != nil {
			return nil, err
		}
		pokemons = append(pokemons, p)
	}
	return pokemons, rows.Err()
}

// buildListQuery constructs a SELECT with optional WHERE clauses.
func buildListQuery(filter pokemon.Filter, page pokemon.Page) (string, []any) {
	base, where, args, n := buildFilterBase(filter)

	q := base
	if len(where) > 0 {
		q += " WHERE " + strings.Join(where, " AND ")
	}
	q += fmt.Sprintf(" ORDER BY p.id LIMIT $%d OFFSET $%d", n, n+1)
	args = append(args, page.Limit, page.Offset)

	return q, args
}

func buildCountQuery(filter pokemon.Filter) (string, []any) {
	_, where, args, _ := buildFilterBase(filter)

	base := `SELECT COUNT(DISTINCT p.id) FROM pokemons p`
	base += buildFilterJoins(filter)

	q := base
	if len(where) > 0 {
		q += " WHERE " + strings.Join(where, " AND ")
	}
	return q, args
}

// buildFilterBase returns the SELECT base with joins, WHERE clauses, args, and next param index.
func buildFilterBase(filter pokemon.Filter) (base string, where []string, args []any, n int) {
	n = 1
	base = `
		SELECT DISTINCT p.id, p.name, p.classification, p.flee_rate, p.max_cp, p.max_hp,
		       p.weight_min, p.weight_max, p.height_min, p.height_max, p.pokemon_class
		FROM pokemons p`
	base += buildFilterJoins(filter)

	if filter.Type != "" {
		where = append(where, fmt.Sprintf("pt.type_name = $%d", n))
		args = append(args, filter.Type)
		n++
	}
	if filter.Weakness != "" {
		where = append(where, fmt.Sprintf("pw.type_name = $%d", n))
		args = append(args, filter.Weakness)
		n++
	}
	if filter.Resistant != "" {
		where = append(where, fmt.Sprintf("pr.type_name = $%d", n))
		args = append(args, filter.Resistant)
		n++
	}
	if filter.Name != "" {
		where = append(where, fmt.Sprintf("LOWER(p.name) LIKE LOWER($%d)", n))
		args = append(args, "%"+filter.Name+"%")
		n++
	}
	if filter.FleeRateMin != nil {
		where = append(where, fmt.Sprintf("p.flee_rate >= $%d", n))
		args = append(args, *filter.FleeRateMin)
		n++
	}
	if filter.FleeRateMax != nil {
		where = append(where, fmt.Sprintf("p.flee_rate <= $%d", n))
		args = append(args, *filter.FleeRateMax)
		n++
	}
	if filter.WeightMin != nil {
		where = append(where, fmt.Sprintf("CAST(REGEXP_REPLACE(p.weight_min, '[^0-9.]', '', 'g') AS DECIMAL) >= $%d", n))
		args = append(args, *filter.WeightMin)
		n++
	}
	if filter.WeightMax != nil {
		where = append(where, fmt.Sprintf("CAST(REGEXP_REPLACE(p.weight_max, '[^0-9.]', '', 'g') AS DECIMAL) <= $%d", n))
		args = append(args, *filter.WeightMax)
		n++
	}
	if filter.HeightMin != nil {
		where = append(where, fmt.Sprintf("CAST(REGEXP_REPLACE(p.height_min, '[^0-9.]', '', 'g') AS DECIMAL) >= $%d", n))
		args = append(args, *filter.HeightMin)
		n++
	}
	if filter.HeightMax != nil {
		where = append(where, fmt.Sprintf("CAST(REGEXP_REPLACE(p.height_max, '[^0-9.]', '', 'g') AS DECIMAL) <= $%d", n))
		args = append(args, *filter.HeightMax)
		n++
	}
	return base, where, args, n
}

// buildFilterJoins returns the JOIN clauses required by the active filters.
func buildFilterJoins(filter pokemon.Filter) string {
	var joins string
	if filter.Type != "" {
		joins += ` INNER JOIN pokemon_types pt ON pt.pokemon_id = p.id`
	}
	if filter.Weakness != "" {
		joins += ` INNER JOIN pokemon_weaknesses pw ON pw.pokemon_id = p.id`
	}
	if filter.Resistant != "" {
		joins += ` INNER JOIN pokemon_resistant pr ON pr.pokemon_id = p.id`
	}
	if filter.Favorite != nil && *filter.Favorite {
		joins += ` INNER JOIN favorites f ON f.pokemon_id = p.id`
	}
	return joins
}
