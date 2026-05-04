package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jstowers/pokemon/internal/handler"
	"github.com/jstowers/pokemon/internal/pokemon"
)

// mockRepo duplicates the mock from the pokemon package tests so the handler
// package has no dependency on test helpers outside its own package.
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

// newTestServer wires a mock repository into a real Service and Handler,
// registers routes, and returns a test server ready to receive requests.
func newTestServer(repo pokemon.Repository) *httptest.Server {
	svc := pokemon.NewService(repo)
	h := handler.New(svc)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux, h)
	return httptest.NewServer(mux)
}

var bulbasaur = &pokemon.Pokemon{
	ID:   "001",
	Name: "Bulbasaur",
	Types: []string{"Grass", "Poison"},
	Attacks: pokemon.Attacks{
		Fast:    []pokemon.Attack{{Name: "Tackle", Type: "Normal", Damage: 12}},
		Special: []pokemon.Attack{{Name: "Seed Bomb", Type: "Grass", Damage: 40}},
	},
}

// ---- GET /pokemons/{id} -----------------------------------------------------

func TestGetByID_OK(t *testing.T) {
	srv := newTestServer(&mockRepo{
		getByID: func(_ context.Context, id string) (*pokemon.Pokemon, error) {
			if id != "001" {
				return nil, pokemon.ErrNotFound
			}
			return bulbasaur, nil
		},
	})
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/pokemons/001")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("got status %d, want 200", resp.StatusCode)
	}

	var body map[string]any
	json.NewDecoder(resp.Body).Decode(&body)
	if body["name"] != "Bulbasaur" {
		t.Errorf("got name %v, want Bulbasaur", body["name"])
	}
}

func TestGetByID_NotFound(t *testing.T) {
	srv := newTestServer(&mockRepo{
		getByID: func(_ context.Context, _ string) (*pokemon.Pokemon, error) {
			return nil, pokemon.ErrNotFound
		},
	})
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/pokemons/999")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("got status %d, want 404", resp.StatusCode)
	}
}

// ---- GET /pokemons ----------------------------------------------------------

func TestList_OK(t *testing.T) {
	srv := newTestServer(&mockRepo{
		list: func(_ context.Context, _ pokemon.Filter, _ pokemon.Page) ([]*pokemon.Pokemon, error) {
			return []*pokemon.Pokemon{bulbasaur}, nil
		},
		count: func(_ context.Context, _ pokemon.Filter) (int, error) {
			return 151, nil
		},
	})
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/pokemons?page=1&limit=20")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("got status %d, want 200", resp.StatusCode)
	}

	var body map[string]any
	json.NewDecoder(resp.Body).Decode(&body)
	if int(body["total"].(float64)) != 151 {
		t.Errorf("got total %v, want 151", body["total"])
	}
}

func TestList_NameFilter(t *testing.T) {
	receivedFilter := pokemon.Filter{}
	srv := newTestServer(&mockRepo{
		list: func(_ context.Context, f pokemon.Filter, _ pokemon.Page) ([]*pokemon.Pokemon, error) {
			receivedFilter = f
			return []*pokemon.Pokemon{bulbasaur}, nil
		},
		count: func(_ context.Context, _ pokemon.Filter) (int, error) { return 1, nil },
	})
	defer srv.Close()

	http.Get(srv.URL + "/pokemons?name=bulba")

	if receivedFilter.Name != "bulba" {
		t.Errorf("got filter.Name %q, want %q", receivedFilter.Name, "bulba")
	}
}

func TestList_TypeFilter(t *testing.T) {
	receivedFilter := pokemon.Filter{}
	srv := newTestServer(&mockRepo{
		list: func(_ context.Context, f pokemon.Filter, _ pokemon.Page) ([]*pokemon.Pokemon, error) {
			receivedFilter = f
			return []*pokemon.Pokemon{bulbasaur}, nil
		},
		count: func(_ context.Context, _ pokemon.Filter) (int, error) { return 1, nil },
	})
	defer srv.Close()

	http.Get(srv.URL + "/pokemons?type=Grass")

	if receivedFilter.Type != "Grass" {
		t.Errorf("got filter.Type %q, want %q", receivedFilter.Type, "Grass")
	}
}

func TestList_WeaknessFilter(t *testing.T) {
	receivedFilter := pokemon.Filter{}
	srv := newTestServer(&mockRepo{
		list: func(_ context.Context, f pokemon.Filter, _ pokemon.Page) ([]*pokemon.Pokemon, error) {
			receivedFilter = f
			return []*pokemon.Pokemon{bulbasaur}, nil
		},
		count: func(_ context.Context, _ pokemon.Filter) (int, error) { return 1, nil },
	})
	defer srv.Close()

	http.Get(srv.URL + "/pokemons?weakness=Fire")

	if receivedFilter.Weakness != "Fire" {
		t.Errorf("got filter.Weakness %q, want %q", receivedFilter.Weakness, "Fire")
	}
}

func TestList_ResistantFilter(t *testing.T) {
	receivedFilter := pokemon.Filter{}
	srv := newTestServer(&mockRepo{
		list: func(_ context.Context, f pokemon.Filter, _ pokemon.Page) ([]*pokemon.Pokemon, error) {
			receivedFilter = f
			return []*pokemon.Pokemon{bulbasaur}, nil
		},
		count: func(_ context.Context, _ pokemon.Filter) (int, error) { return 1, nil },
	})
	defer srv.Close()

	http.Get(srv.URL + "/pokemons?resistant=Water")

	if receivedFilter.Resistant != "Water" {
		t.Errorf("got filter.Resistant %q, want %q", receivedFilter.Resistant, "Water")
	}
}

func TestList_FleeRateFilter(t *testing.T) {
	receivedFilter := pokemon.Filter{}
	srv := newTestServer(&mockRepo{
		list: func(_ context.Context, f pokemon.Filter, _ pokemon.Page) ([]*pokemon.Pokemon, error) {
			receivedFilter = f
			return []*pokemon.Pokemon{bulbasaur}, nil
		},
		count: func(_ context.Context, _ pokemon.Filter) (int, error) { return 1, nil },
	})
	defer srv.Close()

	http.Get(srv.URL + "/pokemons?fleeRateMin=0.1&fleeRateMax=0.5")

	if receivedFilter.FleeRateMin == nil || *receivedFilter.FleeRateMin != 0.1 {
		t.Errorf("got filter.FleeRateMin %v, want 0.1", receivedFilter.FleeRateMin)
	}
	if receivedFilter.FleeRateMax == nil || *receivedFilter.FleeRateMax != 0.5 {
		t.Errorf("got filter.FleeRateMax %v, want 0.5", receivedFilter.FleeRateMax)
	}
}

func TestList_WeightFilter(t *testing.T) {
	receivedFilter := pokemon.Filter{}
	srv := newTestServer(&mockRepo{
		list: func(_ context.Context, f pokemon.Filter, _ pokemon.Page) ([]*pokemon.Pokemon, error) {
			receivedFilter = f
			return []*pokemon.Pokemon{bulbasaur}, nil
		},
		count: func(_ context.Context, _ pokemon.Filter) (int, error) { return 1, nil },
	})
	defer srv.Close()

	http.Get(srv.URL + "/pokemons?weightMin=5&weightMax=50")

	if receivedFilter.WeightMin == nil || *receivedFilter.WeightMin != 5 {
		t.Errorf("got filter.WeightMin %v, want 5", receivedFilter.WeightMin)
	}
	if receivedFilter.WeightMax == nil || *receivedFilter.WeightMax != 50 {
		t.Errorf("got filter.WeightMax %v, want 50", receivedFilter.WeightMax)
	}
}

func TestList_HeightFilter(t *testing.T) {
	receivedFilter := pokemon.Filter{}
	srv := newTestServer(&mockRepo{
		list: func(_ context.Context, f pokemon.Filter, _ pokemon.Page) ([]*pokemon.Pokemon, error) {
			receivedFilter = f
			return []*pokemon.Pokemon{bulbasaur}, nil
		},
		count: func(_ context.Context, _ pokemon.Filter) (int, error) { return 1, nil },
	})
	defer srv.Close()

	http.Get(srv.URL + "/pokemons?heightMin=0.5&heightMax=2.0")

	if receivedFilter.HeightMin == nil || *receivedFilter.HeightMin != 0.5 {
		t.Errorf("got filter.HeightMin %v, want 0.5", receivedFilter.HeightMin)
	}
	if receivedFilter.HeightMax == nil || *receivedFilter.HeightMax != 2.0 {
		t.Errorf("got filter.HeightMax %v, want 2.0", receivedFilter.HeightMax)
	}
}

func TestList_CombinedFilters(t *testing.T) {
	receivedFilter := pokemon.Filter{}
	srv := newTestServer(&mockRepo{
		list: func(_ context.Context, f pokemon.Filter, _ pokemon.Page) ([]*pokemon.Pokemon, error) {
			receivedFilter = f
			return []*pokemon.Pokemon{bulbasaur}, nil
		},
		count: func(_ context.Context, _ pokemon.Filter) (int, error) { return 1, nil },
	})
	defer srv.Close()

	http.Get(srv.URL + "/pokemons?type=Grass&weakness=Fire&fleeRateMax=0.1")

	if receivedFilter.Type != "Grass" {
		t.Errorf("got filter.Type %q, want Grass", receivedFilter.Type)
	}
	if receivedFilter.Weakness != "Fire" {
		t.Errorf("got filter.Weakness %q, want Fire", receivedFilter.Weakness)
	}
	if receivedFilter.FleeRateMax == nil || *receivedFilter.FleeRateMax != 0.1 {
		t.Errorf("got filter.FleeRateMax %v, want 0.1", receivedFilter.FleeRateMax)
	}
}

func TestList_DefaultPagination(t *testing.T) {
	receivedPage := pokemon.Page{}
	srv := newTestServer(&mockRepo{
		list: func(_ context.Context, _ pokemon.Filter, p pokemon.Page) ([]*pokemon.Pokemon, error) {
			receivedPage = p
			return nil, nil
		},
		count: func(_ context.Context, _ pokemon.Filter) (int, error) { return 0, nil },
	})
	defer srv.Close()

	http.Get(srv.URL + "/pokemons")

	if receivedPage.Limit != 20 {
		t.Errorf("got default limit %d, want 20", receivedPage.Limit)
	}
	if receivedPage.Offset != 0 {
		t.Errorf("got default offset %d, want 0", receivedPage.Offset)
	}
}

// ---- POST /favorites --------------------------------------------------------

func TestAddFavorite_OK(t *testing.T) {
	srv := newTestServer(&mockRepo{
		getByID: func(_ context.Context, _ string) (*pokemon.Pokemon, error) {
			return bulbasaur, nil
		},
		isFavorite: func(_ context.Context, _ string) (bool, error) { return false, nil },
		addFavorite: func(_ context.Context, _ string) error { return nil },
	})
	defer srv.Close()

	body := bytes.NewBufferString(`{"pokemonId":"001"}`)
	resp, err := http.Post(srv.URL+"/favorites", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("got status %d, want 204", resp.StatusCode)
	}
}

func TestAddFavorite_MissingID(t *testing.T) {
	srv := newTestServer(&mockRepo{})
	defer srv.Close()

	body := bytes.NewBufferString(`{}`)
	resp, err := http.Post(srv.URL+"/favorites", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("got status %d, want 400", resp.StatusCode)
	}
}

func TestAddFavorite_AlreadyFavorite(t *testing.T) {
	srv := newTestServer(&mockRepo{
		getByID: func(_ context.Context, _ string) (*pokemon.Pokemon, error) {
			return bulbasaur, nil
		},
		isFavorite: func(_ context.Context, _ string) (bool, error) { return true, nil },
	})
	defer srv.Close()

	body := bytes.NewBufferString(`{"pokemonId":"001"}`)
	resp, err := http.Post(srv.URL+"/favorites", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("got status %d, want 409", resp.StatusCode)
	}
}

func TestAddFavorite_PokemonNotFound(t *testing.T) {
	srv := newTestServer(&mockRepo{
		getByID: func(_ context.Context, _ string) (*pokemon.Pokemon, error) {
			return nil, pokemon.ErrNotFound
		},
	})
	defer srv.Close()

	body := bytes.NewBufferString(`{"pokemonId":"999"}`)
	resp, err := http.Post(srv.URL+"/favorites", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("got status %d, want 404", resp.StatusCode)
	}
}

// ---- DELETE /favorites/{id} -------------------------------------------------

func TestRemoveFavorite_OK(t *testing.T) {
	srv := newTestServer(&mockRepo{
		isFavorite: func(_ context.Context, _ string) (bool, error) { return true, nil },
		removeFavorite: func(_ context.Context, _ string) error { return nil },
	})
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/favorites/001", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("got status %d, want 204", resp.StatusCode)
	}
}

func TestRemoveFavorite_NotFound(t *testing.T) {
	srv := newTestServer(&mockRepo{
		isFavorite: func(_ context.Context, _ string) (bool, error) { return false, nil },
	})
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/favorites/001", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("got status %d, want 404", resp.StatusCode)
	}
}

// ---- GET /favorites ---------------------------------------------------------

func TestListFavorites_OK(t *testing.T) {
	srv := newTestServer(&mockRepo{
		listFavorites: func(_ context.Context, _ pokemon.Page) ([]*pokemon.Pokemon, error) {
			return []*pokemon.Pokemon{bulbasaur}, nil
		},
		countFavorites: func(_ context.Context) (int, error) { return 1, nil },
	})
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/favorites")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("got status %d, want 200", resp.StatusCode)
	}

	var body map[string]any
	json.NewDecoder(resp.Body).Decode(&body)
	if int(body["total"].(float64)) != 1 {
		t.Errorf("got total %v, want 1", body["total"])
	}
}

func TestListFavorites_Empty(t *testing.T) {
	srv := newTestServer(&mockRepo{
		listFavorites:  func(_ context.Context, _ pokemon.Page) ([]*pokemon.Pokemon, error) { return nil, nil },
		countFavorites: func(_ context.Context) (int, error) { return 0, nil },
	})
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/favorites")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("got status %d, want 200", resp.StatusCode)
	}
}
