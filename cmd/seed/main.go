package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/jstowers/pokemon/internal/db"
	_ "github.com/lib/pq"
)

func main() {
	cfg := db.Config{
		Host:     getenv("DB_HOST", "localhost"),
		Port:     getenv("DB_PORT", "5432"),
		User:     getenv("DB_USER", "postgres"),
		Password: getenv("DB_PASSWORD", ""),
		DBName:   getenv("DB_NAME", "pokemon"),
		SSLMode:  getenv("DB_SSLMODE", "disable"),
	}

	database, err := db.Open(cfg)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer database.Close()

	if err := db.Migrate(database); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	path := getenv("SEED_FILE", "data/pokemons.json")
	f, err := os.Open(path)
	if err != nil {
		log.Fatalf("open seed file: %v", err)
	}
	defer f.Close()

	var raw []rawPokemon
	if err := json.NewDecoder(f).Decode(&raw); err != nil {
		log.Fatalf("decode json: %v", err)
	}

	if err := seed(database, raw); err != nil {
		log.Fatalf("seed: %v", err)
	}
	log.Printf("seeded %d pokemon", len(raw))
}

func seed(database *sql.DB, pokemons []rawPokemon) error {
	tx, err := database.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, p := range pokemons {
		if err := insertPokemon(tx, p); err != nil {
			return fmt.Errorf("insert %s: %w", p.ID, err)
		}
	}

	return tx.Commit()
}

func insertPokemon(tx *sql.Tx, p rawPokemon) error {
	pokemonClass := ""
	if p.PokemonClass != "" {
		// Extract "LEGENDARY" or "MYTHIC" from the description string.
		switch {
		case contains(p.PokemonClass, "LEGENDARY"):
			pokemonClass = "LEGENDARY"
		case contains(p.PokemonClass, "MYTHIC"):
			pokemonClass = "MYTHIC"
		}
	}

	_, err := tx.Exec(`
		INSERT INTO pokemons (id, name, classification, flee_rate, max_cp, max_hp,
		                      weight_min, weight_max, height_min, height_max, pokemon_class)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (id) DO NOTHING`,
		p.ID, p.Name, p.Classification, p.FleeRate, p.MaxCP, p.MaxHP,
		p.Weight.Minimum, p.Weight.Maximum,
		p.Height.Minimum, p.Height.Maximum,
		pokemonClass,
	)
	if err != nil {
		return err
	}

	if err := insertStringSlice(tx, p.ID, p.Types, "types", "pokemon_types"); err != nil {
		return err
	}
	if err := insertStringSlice(tx, p.ID, p.Resistant, "types", "pokemon_resistant"); err != nil {
		return err
	}
	if err := insertStringSlice(tx, p.ID, p.Weaknesses, "types", "pokemon_weaknesses"); err != nil {
		return err
	}

	for _, a := range p.Attacks.Fast {
		if err := insertAttack(tx, p.ID, a, "fast"); err != nil {
			return err
		}
	}
	for _, a := range p.Attacks.Special {
		if err := insertAttack(tx, p.ID, a, "special"); err != nil {
			return err
		}
	}

	for _, e := range p.Evolutions {
		if _, err := tx.Exec(`
			INSERT INTO evolutions (pokemon_id, evolves_to_id, evolves_to_name, direction)
			VALUES ($1,$2,$3,'next') ON CONFLICT DO NOTHING`,
			p.ID, e.ID, e.Name); err != nil {
			return err
		}
	}
	for _, e := range p.PrevEvolutions {
		if _, err := tx.Exec(`
			INSERT INTO evolutions (pokemon_id, evolves_to_id, evolves_to_name, direction)
			VALUES ($1,$2,$3,'prev') ON CONFLICT DO NOTHING`,
			p.ID, e.ID, e.Name); err != nil {
			return err
		}
	}

	if p.EvolutionReq != nil {
		if _, err := tx.Exec(`
			INSERT INTO evolution_requirements (pokemon_id, amount, item_name)
			VALUES ($1,$2,$3) ON CONFLICT DO NOTHING`,
			p.ID, p.EvolutionReq.Amount, p.EvolutionReq.Name); err != nil {
			return err
		}
	}

	return nil
}

func insertStringSlice(tx *sql.Tx, pokemonID string, values []string, typeTable, junctionTable string) error {
	for _, v := range values {
		if _, err := tx.Exec(
			fmt.Sprintf(`INSERT INTO %s (name) VALUES ($1) ON CONFLICT DO NOTHING`, typeTable), v,
		); err != nil {
			return err
		}
		if _, err := tx.Exec(
			fmt.Sprintf(`INSERT INTO %s (pokemon_id, type_name) VALUES ($1,$2) ON CONFLICT DO NOTHING`, junctionTable),
			pokemonID, v,
		); err != nil {
			return err
		}
	}
	return nil
}

func insertAttack(tx *sql.Tx, pokemonID string, a rawAttack, category string) error {
	var attackID int
	err := tx.QueryRow(`
		INSERT INTO attacks (name, type, damage, category)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT DO NOTHING
		RETURNING id`,
		a.Name, a.Type, a.Damage, category,
	).Scan(&attackID)

	if err == sql.ErrNoRows {
		// Already exists; fetch the ID.
		err = tx.QueryRow(
			`SELECT id FROM attacks WHERE name=$1 AND category=$2`, a.Name, category,
		).Scan(&attackID)
	}
	if err != nil {
		return err
	}

	_, err = tx.Exec(
		`INSERT INTO pokemon_attacks (pokemon_id, attack_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`,
		pokemonID, attackID,
	)
	return err
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// ---- raw JSON types (match the shape of pokemons.json) ---------------------

type rawPokemon struct {
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	Classification string          `json:"classification"`
	Types          []string        `json:"types"`
	Resistant      []string        `json:"resistant"`
	Weaknesses     []string        `json:"weaknesses"`
	Weight         rawRange        `json:"weight"`
	Height         rawRange        `json:"height"`
	FleeRate       float64         `json:"fleeRate"`
	MaxCP          int             `json:"maxCP"`
	MaxHP          int             `json:"maxHP"`
	Attacks        rawAttacks      `json:"attacks"`
	Evolutions     []rawEvolution  `json:"evolutions"`
	PrevEvolutions []rawEvolution  `json:"Previous evolution(s)"`
	EvolutionReq   *rawEvolutionReq `json:"evolutionRequirements"`
	PokemonClass   string          `json:"Pokémon Class"`
}

type rawRange struct {
	Minimum string `json:"minimum"`
	Maximum string `json:"maximum"`
}

type rawAttacks struct {
	Fast    []rawAttack `json:"fast"`
	Special []rawAttack `json:"special"`
}

type rawAttack struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Damage int    `json:"damage"`
}

type rawEvolution struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type rawEvolutionReq struct {
	Amount int    `json:"amount"`
	Name   string `json:"name"`
}
