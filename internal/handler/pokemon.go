package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/jstowers/pokemon/internal/pokemon"
)

// Handler holds dependencies for all HTTP handlers.
type Handler struct {
	svc *pokemon.Service
}

// New constructs a Handler.
func New(svc *pokemon.Service) *Handler {
	return &Handler{svc: svc}
}

// ---- response types ---------------------------------------------------------

// PokemonResponse is the JSON representation of a Pokemon.
type PokemonResponse struct {
	ID             string              `json:"id"`
	Name           string              `json:"name"`
	Classification string              `json:"classification"`
	Types          []string            `json:"types"`
	Resistant      []string            `json:"resistant"`
	Weaknesses     []string            `json:"weaknesses"`
	Weight         RangeResponse       `json:"weight"`
	Height         RangeResponse       `json:"height"`
	FleeRate       float64             `json:"fleeRate"`
	MaxCP          int                 `json:"maxCP"`
	MaxHP          int                 `json:"maxHP"`
	Attacks        AttacksResponse     `json:"attacks"`
	Evolutions     []EvolutionResponse `json:"evolutions,omitempty"`
	PrevEvolutions []EvolutionResponse `json:"previousEvolutions,omitempty"`
	EvolutionReq   *EvolutionReqResp   `json:"evolutionRequirements,omitempty"`
	PokemonClass   string              `json:"pokemonClass,omitempty"`
}

// RangeResponse holds minimum and maximum string values.
type RangeResponse struct {
	Minimum string `json:"minimum"`
	Maximum string `json:"maximum"`
}

// AttacksResponse groups fast and special attacks.
type AttacksResponse struct {
	Fast    []AttackResponse `json:"fast"`
	Special []AttackResponse `json:"special"`
}

// AttackResponse is a single move.
type AttackResponse struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Damage int    `json:"damage"`
}

// EvolutionResponse is a reference to another Pokemon in the evolution chain.
type EvolutionResponse struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// EvolutionReqResp describes what is needed to evolve.
type EvolutionReqResp struct {
	Amount int    `json:"amount"`
	Name   string `json:"name"`
}

// ListResponse is a paginated list of Pokemon.
type ListResponse struct {
	Data       []*PokemonResponse `json:"data"`
	Total      int                `json:"total"`
	Page       int                `json:"page"`
	Limit      int                `json:"limit"`
	TotalPages int                `json:"totalPages"`
}

// ErrorResponse is returned on failure.
type ErrorResponse struct {
	Error string `json:"error"`
}

// AddFavoriteRequest is the body for POST /favorites.
type AddFavoriteRequest struct {
	PokemonID string `json:"pokemonId"`
}

// ---- handlers ---------------------------------------------------------------

// GetByID godoc
//
//	@Summary		Get a Pokemon by ID
//	@Tags			pokemon
//	@Produce		json
//	@Param			id	path		string	true	"Pokemon ID (e.g. 001)"
//	@Success		200	{object}	PokemonResponse
//	@Failure		404	{object}	ErrorResponse
//	@Router			/pokemons/{id} [get]
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	p, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toResponse(p))
}

// List godoc
//
//	@Summary		List Pokemon
//	@Description	Returns a paginated list of Pokemon. Supports filtering by name (partial match) and type.
//	@Tags			pokemon
//	@Produce		json
//	@Param			name	query		string	false	"Filter by name (partial, case-insensitive)"
//	@Param			type	query		string	false	"Filter by type (e.g. Fire, Water)"
//	@Param			page	query		int		false	"Page number (default 1)"
//	@Param			limit	query		int		false	"Results per page, max 100 (default 20)"
//	@Success		200		{object}	ListResponse
//	@Router			/pokemons [get]
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	filter := pokemon.Filter{
		Name: r.URL.Query().Get("name"),
		Type: r.URL.Query().Get("type"),
	}

	page, limit := parsePagination(r)

	pokemons, total, err := h.svc.List(r.Context(), filter, pokemon.Page{
		Limit:  limit,
		Offset: (page - 1) * limit,
	})
	if err != nil {
		writeError(w, err)
		return
	}

	resp := ListResponse{
		Data:       toResponses(pokemons),
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages(total, limit),
	}
	writeJSON(w, http.StatusOK, resp)
}

// AddFavorite godoc
//
//	@Summary		Add a Pokemon to favorites
//	@Tags			favorites
//	@Accept			json
//	@Produce		json
//	@Param			body	body	AddFavoriteRequest	true	"Pokemon to favorite"
//	@Success		204
//	@Failure		400	{object}	ErrorResponse	"Missing or invalid pokemonId"
//	@Failure		404	{object}	ErrorResponse	"Pokemon not found"
//	@Failure		409	{object}	ErrorResponse	"Already a favorite"
//	@Router			/favorites [post]
func (h *Handler) AddFavorite(w http.ResponseWriter, r *http.Request) {
	var body AddFavoriteRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{"invalid request body"})
		return
	}
	if body.PokemonID == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{"pokemonId is required"})
		return
	}

	if err := h.svc.AddFavorite(r.Context(), body.PokemonID); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// RemoveFavorite godoc
//
//	@Summary		Remove a Pokemon from favorites
//	@Tags			favorites
//	@Produce		json
//	@Param			id	path	string	true	"Pokemon ID"
//	@Success		204
//	@Failure		404	{object}	ErrorResponse	"Not a favorite"
//	@Router			/favorites/{id} [delete]
func (h *Handler) RemoveFavorite(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.svc.RemoveFavorite(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListFavorites godoc
//
//	@Summary		List favorite Pokemon
//	@Tags			favorites
//	@Produce		json
//	@Param			page	query		int	false	"Page number (default 1)"
//	@Param			limit	query		int	false	"Results per page, max 100 (default 20)"
//	@Success		200		{object}	ListResponse
//	@Router			/favorites [get]
func (h *Handler) ListFavorites(w http.ResponseWriter, r *http.Request) {
	page, limit := parsePagination(r)

	pokemons, total, err := h.svc.ListFavorites(r.Context(), pokemon.Page{
		Limit:  limit,
		Offset: (page - 1) * limit,
	})
	if err != nil {
		writeError(w, err)
		return
	}

	resp := ListResponse{
		Data:       toResponses(pokemons),
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages(total, limit),
	}
	writeJSON(w, http.StatusOK, resp)
}

// ---- helpers ----------------------------------------------------------------

func parsePagination(r *http.Request) (page, limit int) {
	page, _ = strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ = strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 20
	}
	return page, limit
}

func totalPages(total, limit int) int {
	if limit == 0 {
		return 0
	}
	pages := total / limit
	if total%limit > 0 {
		pages++
	}
	return pages
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, pokemon.ErrNotFound):
		writeJSON(w, http.StatusNotFound, ErrorResponse{err.Error()})
	case errors.Is(err, pokemon.ErrAlreadyFavorite):
		writeJSON(w, http.StatusConflict, ErrorResponse{err.Error()})
	case errors.Is(err, pokemon.ErrNotFavorite):
		writeJSON(w, http.StatusNotFound, ErrorResponse{err.Error()})
	default:
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{"internal server error"})
	}
}

func toResponse(p *pokemon.Pokemon) *PokemonResponse {
	r := &PokemonResponse{
		ID:             p.ID,
		Name:           p.Name,
		Classification: p.Classification,
		Types:          p.Types,
		Resistant:      p.Resistant,
		Weaknesses:     p.Weaknesses,
		Weight:         RangeResponse(p.Weight),
		Height:         RangeResponse(p.Height),
		FleeRate:       p.FleeRate,
		MaxCP:          p.MaxCP,
		MaxHP:          p.MaxHP,
		PokemonClass:   p.PokemonClass,
	}

	for _, a := range p.Attacks.Fast {
		r.Attacks.Fast = append(r.Attacks.Fast, AttackResponse(a))
	}
	for _, a := range p.Attacks.Special {
		r.Attacks.Special = append(r.Attacks.Special, AttackResponse(a))
	}
	for _, e := range p.Evolutions {
		r.Evolutions = append(r.Evolutions, EvolutionResponse(e))
	}
	for _, e := range p.PrevEvolutions {
		r.PrevEvolutions = append(r.PrevEvolutions, EvolutionResponse(e))
	}
	if p.EvolutionReq != nil {
		r.EvolutionReq = &EvolutionReqResp{Amount: p.EvolutionReq.Amount, Name: p.EvolutionReq.Name}
	}

	return r
}

func toResponses(pokemons []*pokemon.Pokemon) []*PokemonResponse {
	out := make([]*PokemonResponse, len(pokemons))
	for i, p := range pokemons {
		out[i] = toResponse(p)
	}
	return out
}
