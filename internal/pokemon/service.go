package pokemon

import (
	"context"
	"errors"
	"fmt"
)

// ErrNotFound is returned when a Pokemon does not exist.
var ErrNotFound = errors.New("pokemon not found")

// ErrAlreadyFavorite is returned when a Pokemon is already favorited.
var ErrAlreadyFavorite = errors.New("pokemon is already a favorite")

// ErrNotFavorite is returned when removing a Pokemon that is not favorited.
var ErrNotFavorite = errors.New("pokemon is not a favorite")

// Service implements all business logic for Pokemon and favorites.
// It depends only on the Repository interface — no database or HTTP details.
type Service struct {
	repo Repository
}

// NewService constructs a Service with the given Repository.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// GetByID returns a Pokemon by its string ID (e.g. "001").
func (s *Service) GetByID(ctx context.Context, id string) (*Pokemon, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("GetByID %s: %w", id, err)
	}
	return p, nil
}

// GetByName returns a Pokemon by exact name (case-insensitive).
func (s *Service) GetByName(ctx context.Context, name string) (*Pokemon, error) {
	p, err := s.repo.GetByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("GetByName %q: %w", name, err)
	}
	return p, nil
}

// List returns a paginated, filtered list of Pokemon.
func (s *Service) List(ctx context.Context, filter Filter, page Page) ([]*Pokemon, int, error) {
	pokemons, err := s.repo.List(ctx, filter, page)
	if err != nil {
		return nil, 0, fmt.Errorf("List: %w", err)
	}
	total, err := s.repo.Count(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("Count: %w", err)
	}
	return pokemons, total, nil
}

// AddFavorite marks a Pokemon as a favorite.
func (s *Service) AddFavorite(ctx context.Context, pokemonID string) error {
	// Verify the Pokemon exists.
	_, err := s.repo.GetByID(ctx, pokemonID)
	if err != nil {
		return fmt.Errorf("AddFavorite: %w", err)
	}

	already, err := s.repo.IsFavorite(ctx, pokemonID)
	if err != nil {
		return fmt.Errorf("AddFavorite: %w", err)
	}
	if already {
		return ErrAlreadyFavorite
	}

	return s.repo.AddFavorite(ctx, pokemonID)
}

// RemoveFavorite removes a Pokemon from favorites.
func (s *Service) RemoveFavorite(ctx context.Context, pokemonID string) error {
	favorite, err := s.repo.IsFavorite(ctx, pokemonID)
	if err != nil {
		return fmt.Errorf("RemoveFavorite: %w", err)
	}
	if !favorite {
		return ErrNotFavorite
	}

	return s.repo.RemoveFavorite(ctx, pokemonID)
}

// ListFavorites returns a paginated list of favorited Pokemon.
func (s *Service) ListFavorites(ctx context.Context, page Page) ([]*Pokemon, int, error) {
	pokemons, err := s.repo.ListFavorites(ctx, page)
	if err != nil {
		return nil, 0, fmt.Errorf("ListFavorites: %w", err)
	}
	total, err := s.repo.CountFavorites(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("CountFavorites: %w", err)
	}
	return pokemons, total, nil
}
