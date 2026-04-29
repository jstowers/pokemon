package pokemon

import "context"

// Repository defines all database operations for Pokemon and favorites.
// The PostgreSQL implementation lives in internal/repository.
type Repository interface {
	// Pokemon queries
	GetByID(ctx context.Context, id string) (*Pokemon, error)
	GetByName(ctx context.Context, name string) (*Pokemon, error)
	List(ctx context.Context, filter Filter, page Page) ([]*Pokemon, error)
	Count(ctx context.Context, filter Filter) (int, error)

	// Favorites
	AddFavorite(ctx context.Context, pokemonID string) error
	RemoveFavorite(ctx context.Context, pokemonID string) error
	ListFavorites(ctx context.Context, page Page) ([]*Pokemon, error)
	CountFavorites(ctx context.Context) (int, error)
	IsFavorite(ctx context.Context, pokemonID string) (bool, error)
}
