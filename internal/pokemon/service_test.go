package pokemon_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jstowers/pokemon/internal/pokemon"
)

// mockRepo implements pokemon.Repository using function fields.
// Each test sets only the functions it needs; unset functions panic if called,
// which immediately signals an unexpected call.
type mockRepo struct {
	getByID        func(ctx context.Context, id string) (*pokemon.Pokemon, error)
	getByName      func(ctx context.Context, name string) (*pokemon.Pokemon, error)
	list           func(ctx context.Context, filter pokemon.Filter, page pokemon.Page) ([]*pokemon.Pokemon, error)
	count          func(ctx context.Context, filter pokemon.Filter) (int, error)
	addFavorite    func(ctx context.Context, pokemonID string) error
	removeFavorite func(ctx context.Context, pokemonID string) error
	listFavorites  func(ctx context.Context, page pokemon.Page) ([]*pokemon.Pokemon, error)
	countFavorites func(ctx context.Context) (int, error)
	isFavorite     func(ctx context.Context, pokemonID string) (bool, error)
}

func (m *mockRepo) GetByID(ctx context.Context, id string) (*pokemon.Pokemon, error) {
	return m.getByID(ctx, id)
}
func (m *mockRepo) GetByName(ctx context.Context, name string) (*pokemon.Pokemon, error) {
	return m.getByName(ctx, name)
}
func (m *mockRepo) List(ctx context.Context, f pokemon.Filter, p pokemon.Page) ([]*pokemon.Pokemon, error) {
	return m.list(ctx, f, p)
}
func (m *mockRepo) Count(ctx context.Context, f pokemon.Filter) (int, error) {
	return m.count(ctx, f)
}
func (m *mockRepo) AddFavorite(ctx context.Context, id string) error    { return m.addFavorite(ctx, id) }
func (m *mockRepo) RemoveFavorite(ctx context.Context, id string) error { return m.removeFavorite(ctx, id) }
func (m *mockRepo) ListFavorites(ctx context.Context, p pokemon.Page) ([]*pokemon.Pokemon, error) {
	return m.listFavorites(ctx, p)
}
func (m *mockRepo) CountFavorites(ctx context.Context) (int, error) { return m.countFavorites(ctx) }
func (m *mockRepo) IsFavorite(ctx context.Context, id string) (bool, error) {
	return m.isFavorite(ctx, id)
}

// bulbasaur is a minimal Pokemon fixture used across tests.
var bulbasaur = &pokemon.Pokemon{ID: "001", Name: "Bulbasaur"}

// ---- GetByID ----------------------------------------------------------------

func TestGetByID_Found(t *testing.T) {
	svc := pokemon.NewService(&mockRepo{
		getByID: func(_ context.Context, id string) (*pokemon.Pokemon, error) {
			if id != "001" {
				t.Fatalf("unexpected id: %s", id)
			}
			return bulbasaur, nil
		},
	})

	p, err := svc.GetByID(context.Background(), "001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name != "Bulbasaur" {
		t.Errorf("got name %q, want %q", p.Name, "Bulbasaur")
	}
}

func TestGetByID_NotFound(t *testing.T) {
	svc := pokemon.NewService(&mockRepo{
		getByID: func(_ context.Context, _ string) (*pokemon.Pokemon, error) {
			return nil, pokemon.ErrNotFound
		},
	})

	_, err := svc.GetByID(context.Background(), "999")
	if !errors.Is(err, pokemon.ErrNotFound) {
		t.Errorf("got %v, want ErrNotFound", err)
	}
}

// ---- GetByName --------------------------------------------------------------

func TestGetByName_Found(t *testing.T) {
	svc := pokemon.NewService(&mockRepo{
		getByName: func(_ context.Context, name string) (*pokemon.Pokemon, error) {
			if name != "Bulbasaur" {
				t.Fatalf("unexpected name: %s", name)
			}
			return bulbasaur, nil
		},
	})

	p, err := svc.GetByName(context.Background(), "Bulbasaur")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.ID != "001" {
		t.Errorf("got id %q, want %q", p.ID, "001")
	}
}

func TestGetByName_NotFound(t *testing.T) {
	svc := pokemon.NewService(&mockRepo{
		getByName: func(_ context.Context, _ string) (*pokemon.Pokemon, error) {
			return nil, pokemon.ErrNotFound
		},
	})

	_, err := svc.GetByName(context.Background(), "Unknown")
	if !errors.Is(err, pokemon.ErrNotFound) {
		t.Errorf("got %v, want ErrNotFound", err)
	}
}

// ---- List -------------------------------------------------------------------

func TestList(t *testing.T) {
	want := []*pokemon.Pokemon{bulbasaur}
	svc := pokemon.NewService(&mockRepo{
		list: func(_ context.Context, _ pokemon.Filter, _ pokemon.Page) ([]*pokemon.Pokemon, error) {
			return want, nil
		},
		count: func(_ context.Context, _ pokemon.Filter) (int, error) {
			return 151, nil
		},
	})

	pokemons, total, err := svc.List(context.Background(), pokemon.Filter{}, pokemon.Page{Limit: 20})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 151 {
		t.Errorf("got total %d, want 151", total)
	}
	if len(pokemons) != 1 {
		t.Errorf("got %d pokemons, want 1", len(pokemons))
	}
}

// ---- AddFavorite ------------------------------------------------------------

func TestAddFavorite_Success(t *testing.T) {
	added := false
	svc := pokemon.NewService(&mockRepo{
		getByID: func(_ context.Context, _ string) (*pokemon.Pokemon, error) {
			return bulbasaur, nil
		},
		isFavorite: func(_ context.Context, _ string) (bool, error) {
			return false, nil
		},
		addFavorite: func(_ context.Context, _ string) error {
			added = true
			return nil
		},
	})

	if err := svc.AddFavorite(context.Background(), "001"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !added {
		t.Error("expected AddFavorite to be called on repository")
	}
}

func TestAddFavorite_PokemonNotFound(t *testing.T) {
	svc := pokemon.NewService(&mockRepo{
		getByID: func(_ context.Context, _ string) (*pokemon.Pokemon, error) {
			return nil, pokemon.ErrNotFound
		},
	})

	err := svc.AddFavorite(context.Background(), "999")
	if !errors.Is(err, pokemon.ErrNotFound) {
		t.Errorf("got %v, want ErrNotFound", err)
	}
}

func TestAddFavorite_AlreadyFavorite(t *testing.T) {
	svc := pokemon.NewService(&mockRepo{
		getByID: func(_ context.Context, _ string) (*pokemon.Pokemon, error) {
			return bulbasaur, nil
		},
		isFavorite: func(_ context.Context, _ string) (bool, error) {
			return true, nil
		},
	})

	err := svc.AddFavorite(context.Background(), "001")
	if !errors.Is(err, pokemon.ErrAlreadyFavorite) {
		t.Errorf("got %v, want ErrAlreadyFavorite", err)
	}
}

// ---- RemoveFavorite ---------------------------------------------------------

func TestRemoveFavorite_Success(t *testing.T) {
	removed := false
	svc := pokemon.NewService(&mockRepo{
		isFavorite: func(_ context.Context, _ string) (bool, error) {
			return true, nil
		},
		removeFavorite: func(_ context.Context, _ string) error {
			removed = true
			return nil
		},
	})

	if err := svc.RemoveFavorite(context.Background(), "001"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !removed {
		t.Error("expected RemoveFavorite to be called on repository")
	}
}

func TestRemoveFavorite_NotFavorite(t *testing.T) {
	svc := pokemon.NewService(&mockRepo{
		isFavorite: func(_ context.Context, _ string) (bool, error) {
			return false, nil
		},
	})

	err := svc.RemoveFavorite(context.Background(), "001")
	if !errors.Is(err, pokemon.ErrNotFavorite) {
		t.Errorf("got %v, want ErrNotFavorite", err)
	}
}

// ---- ListFavorites ----------------------------------------------------------

func TestListFavorites(t *testing.T) {
	want := []*pokemon.Pokemon{bulbasaur}
	svc := pokemon.NewService(&mockRepo{
		listFavorites: func(_ context.Context, _ pokemon.Page) ([]*pokemon.Pokemon, error) {
			return want, nil
		},
		countFavorites: func(_ context.Context) (int, error) {
			return 1, nil
		},
	})

	pokemons, total, err := svc.ListFavorites(context.Background(), pokemon.Page{Limit: 20})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 {
		t.Errorf("got total %d, want 1", total)
	}
	if len(pokemons) != 1 || pokemons[0].Name != "Bulbasaur" {
		t.Errorf("unexpected pokemons: %v", pokemons)
	}
}
