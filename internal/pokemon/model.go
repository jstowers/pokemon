package pokemon

// Pokemon represents the core domain entity.
type Pokemon struct {
	ID               string
	Name             string
	Classification   string
	Types            []string
	Resistant        []string
	Weaknesses       []string
	Weight           Range
	Height           Range
	FleeRate         float64
	MaxCP            int
	MaxHP            int
	Attacks          Attacks
	Evolutions       []Evolution
	PrevEvolutions   []Evolution
	EvolutionReq     *EvolutionRequirement
	PokemonClass     string // "LEGENDARY" or "MYTHIC", empty if neither
}

// Range holds a minimum/maximum string pair (e.g. "6.04kg" / "7.76kg").
type Range struct {
	Minimum string
	Maximum string
}

// Attacks groups fast and special attacks.
type Attacks struct {
	Fast    []Attack
	Special []Attack
}

// Attack represents a single move.
type Attack struct {
	Name   string
	Type   string
	Damage int
}

// Evolution is a reference to another Pokemon in the evolution chain.
type Evolution struct {
	ID   int
	Name string
}

// EvolutionRequirement describes what is needed to evolve.
type EvolutionRequirement struct {
	Amount int
	Name   string
}

// Filter holds optional query parameters for listing Pokemon.
type Filter struct {
	Name       string
	Type       string
	Favorite   *bool // nil = no filter, true = favorites only
}

// Page holds pagination parameters.
type Page struct {
	Limit  int
	Offset int
}
