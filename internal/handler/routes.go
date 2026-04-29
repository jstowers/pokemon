package handler

import "net/http"

// RegisterRoutes attaches all API routes to the given mux.
func RegisterRoutes(mux *http.ServeMux, h *Handler) {
	mux.HandleFunc("GET /pokemons", h.List)
	mux.HandleFunc("GET /pokemons/{id}", h.GetByID)
	mux.HandleFunc("GET /favorites", h.ListFavorites)
	mux.HandleFunc("POST /favorites", h.AddFavorite)
	mux.HandleFunc("DELETE /favorites/{id}", h.RemoveFavorite)
}
