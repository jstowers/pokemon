//go:build integration

package repository_test

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"os"
	"testing"

	"github.com/jstowers/pokemon/internal/db"
	"github.com/jstowers/pokemon/internal/pokemon"
	"github.com/jstowers/pokemon/internal/repository"
)

var (
	testDB   *sql.DB
	testRepo pokemon.Repository
)

// TestMain connects to pokemon_test, migrates, seeds fixtures, runs all tests,
// then truncates the fixture data so the next run starts clean.
func TestMain(m *testing.M) {
	var err error
	testDB, err = db.Open(db.Config{
		Host:     envOr("TEST_DB_HOST", "localhost"),
		Port:     envOr("TEST_DB_PORT", "5432"),
		User:     envOr("TEST_DB_USER", os.Getenv("USER")),
		Password: envOr("TEST_DB_PASSWORD", ""),
		DBName:   envOr("TEST_DB_NAME", "pokemon_test"),
		SSLMode:  envOr("TEST_DB_SSLMODE", "disable"),
	})
	if err != nil {
		log.Fatalf("open test DB: %v\n\nEnsure pokemon_test exists: createdb pokemon_test", err)
	}
	defer testDB.Close()

	if err := db.Migrate(testDB); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	if err := insertFixtures(testDB); err != nil {
		log.Fatalf("insert fixtures: %v", err)
	}

	testRepo = repository.New(testDB)

	code := m.Run()
	cleanAll(testDB)
	os.Exit(code)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// insertFixtures seeds a minimal set of Pokemon used across all tests.
// ON CONFLICT DO NOTHING makes it idempotent if TestMain is called more than once.
func insertFixtures(database *sql.DB) error {
	tx, err := database.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	fixtures := []struct {
		id, name, class string
		types           []string
	}{
		{"001", "Bulbasaur", "Seed Pokemon", []string{"Grass", "Poison"}},
		{"004", "Charmander", "Lizard Pokemon", []string{"Fire"}},
		{"025", "Pikachu", "Mouse Pokemon", []string{"Electric"}},
	}

	for _, f := range fixtures {
		if _, err := tx.Exec(`
			INSERT INTO pokemons
				(id, name, classification, flee_rate, max_cp, max_hp,
				 weight_min, weight_max, height_min, height_max, pokemon_class)
			VALUES ($1,$2,$3,0.1,1000,100,'1.0kg','2.0kg','0.5m','1.0m','')
			ON CONFLICT DO NOTHING`,
			f.id, f.name, f.class,
		); err != nil {
			return err
		}
		for _, t := range f.types {
			if _, err := tx.Exec(
				`INSERT INTO types (name) VALUES ($1) ON CONFLICT DO NOTHING`, t,
			); err != nil {
				return err
			}
			if _, err := tx.Exec(
				`INSERT INTO pokemon_types (pokemon_id, type_name) VALUES ($1,$2) ON CONFLICT DO NOTHING`,
				f.id, t,
			); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

// cleanAll truncates all fixture data, cascading to junction tables.
func cleanAll(database *sql.DB) {
	database.Exec(`TRUNCATE pokemons, types, attacks CASCADE`)
}

// cleanFavorites registers a cleanup that deletes all favorites rows after each test.
func cleanFavorites(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		if _, err := testDB.Exec(`DELETE FROM favorites`); err != nil {
			t.Logf("cleanup favorites: %v", err)
		}
	})
}

// ---- GetByID ----------------------------------------------------------------

func TestGetByID(t *testing.T) {
	p, err := testRepo.GetByID(context.Background(), "001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name != "Bulbasaur" {
		t.Errorf("got name %q, want Bulbasaur", p.Name)
	}
	if len(p.Types) != 2 {
		t.Errorf("got %d types, want 2", len(p.Types))
	}
}

func TestGetByID_NotFound(t *testing.T) {
	_, err := testRepo.GetByID(context.Background(), "999")
	if !errors.Is(err, pokemon.ErrNotFound) {
		t.Errorf("got %v, want ErrNotFound", err)
	}
}

// ---- GetByName --------------------------------------------------------------

func TestGetByName(t *testing.T) {
	p, err := testRepo.GetByName(context.Background(), "Bulbasaur")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.ID != "001" {
		t.Errorf("got ID %q, want 001", p.ID)
	}
}

func TestGetByName_CaseInsensitive(t *testing.T) {
	p, err := testRepo.GetByName(context.Background(), "bulbasaur")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.ID != "001" {
		t.Errorf("got ID %q, want 001", p.ID)
	}
}

func TestGetByName_NotFound(t *testing.T) {
	_, err := testRepo.GetByName(context.Background(), "Missingno")
	if !errors.Is(err, pokemon.ErrNotFound) {
		t.Errorf("got %v, want ErrNotFound", err)
	}
}

// ---- List -------------------------------------------------------------------

func TestList_All(t *testing.T) {
	results, err := testRepo.List(context.Background(), pokemon.Filter{}, pokemon.Page{Limit: 20})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("got %d results, want 3", len(results))
	}
}

func TestList_TypeFilter(t *testing.T) {
	results, err := testRepo.List(context.Background(), pokemon.Filter{Type: "Grass"}, pokemon.Page{Limit: 20})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("got %d results, want 1", len(results))
	}
	if results[0].Name != "Bulbasaur" {
		t.Errorf("got %q, want Bulbasaur", results[0].Name)
	}
}

func TestList_NameFilter(t *testing.T) {
	results, err := testRepo.List(context.Background(), pokemon.Filter{Name: "char"}, pokemon.Page{Limit: 20})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("got %d results, want 1", len(results))
	}
	if results[0].Name != "Charmander" {
		t.Errorf("got %q, want Charmander", results[0].Name)
	}
}

func TestList_Pagination(t *testing.T) {
	page1, err := testRepo.List(context.Background(), pokemon.Filter{}, pokemon.Page{Limit: 2, Offset: 0})
	if err != nil {
		t.Fatalf("page 1: %v", err)
	}
	if len(page1) != 2 {
		t.Errorf("page 1: got %d, want 2", len(page1))
	}

	page2, err := testRepo.List(context.Background(), pokemon.Filter{}, pokemon.Page{Limit: 2, Offset: 2})
	if err != nil {
		t.Fatalf("page 2: %v", err)
	}
	if len(page2) != 1 {
		t.Errorf("page 2: got %d, want 1", len(page2))
	}
}

// ---- Count ------------------------------------------------------------------

func TestCount_All(t *testing.T) {
	total, err := testRepo.Count(context.Background(), pokemon.Filter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 3 {
		t.Errorf("got %d, want 3", total)
	}
}

func TestCount_TypeFilter(t *testing.T) {
	total, err := testRepo.Count(context.Background(), pokemon.Filter{Type: "Fire"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 {
		t.Errorf("got %d, want 1", total)
	}
}

// ---- Favorites --------------------------------------------------------------

func TestIsFavorite_False(t *testing.T) {
	cleanFavorites(t)

	isFav, err := testRepo.IsFavorite(context.Background(), "001")
	if err != nil {
		t.Fatalf("IsFavorite: %v", err)
	}
	if isFav {
		t.Error("expected false before any favorites added")
	}
}

func TestAddFavorite(t *testing.T) {
	cleanFavorites(t)

	if err := testRepo.AddFavorite(context.Background(), "001"); err != nil {
		t.Fatalf("AddFavorite: %v", err)
	}

	isFav, err := testRepo.IsFavorite(context.Background(), "001")
	if err != nil {
		t.Fatalf("IsFavorite: %v", err)
	}
	if !isFav {
		t.Error("expected IsFavorite to return true after adding")
	}
}

func TestRemoveFavorite(t *testing.T) {
	cleanFavorites(t)

	if err := testRepo.AddFavorite(context.Background(), "001"); err != nil {
		t.Fatalf("AddFavorite: %v", err)
	}
	if err := testRepo.RemoveFavorite(context.Background(), "001"); err != nil {
		t.Fatalf("RemoveFavorite: %v", err)
	}

	isFav, err := testRepo.IsFavorite(context.Background(), "001")
	if err != nil {
		t.Fatalf("IsFavorite: %v", err)
	}
	if isFav {
		t.Error("expected IsFavorite to return false after removal")
	}
}

func TestListFavorites(t *testing.T) {
	cleanFavorites(t)

	for _, id := range []string{"001", "004"} {
		if err := testRepo.AddFavorite(context.Background(), id); err != nil {
			t.Fatalf("AddFavorite %s: %v", id, err)
		}
	}

	results, err := testRepo.ListFavorites(context.Background(), pokemon.Page{Limit: 20})
	if err != nil {
		t.Fatalf("ListFavorites: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("got %d favorites, want 2", len(results))
	}
}

func TestCountFavorites(t *testing.T) {
	cleanFavorites(t)

	for _, id := range []string{"001", "004", "025"} {
		if err := testRepo.AddFavorite(context.Background(), id); err != nil {
			t.Fatalf("AddFavorite %s: %v", id, err)
		}
	}

	total, err := testRepo.CountFavorites(context.Background())
	if err != nil {
		t.Fatalf("CountFavorites: %v", err)
	}
	if total != 3 {
		t.Errorf("got %d, want 3", total)
	}
}
