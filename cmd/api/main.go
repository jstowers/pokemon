// Package main is the entry point for the Pokemon API server.
//
//	@title			Pokemon API
//	@version		1.0
//	@description	REST API for browsing and favoriting Generation I Pokemon.
//	@host			localhost:8080
//	@BasePath		/
package main

import (
	"log"
	"net/http"
	"os"

	_ "github.com/jstowers/pokemon/docs"
	"github.com/jstowers/pokemon/internal/db"
	"github.com/jstowers/pokemon/internal/handler"
	"github.com/jstowers/pokemon/internal/pokemon"
	"github.com/jstowers/pokemon/internal/repository"
	httpSwagger "github.com/swaggo/http-swagger/v2"
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

	repo := repository.New(database)
	svc := pokemon.NewService(repo)
	h := handler.New(svc)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux, h)
	mux.HandleFunc("GET /swagger/", httpSwagger.WrapHandler)

	addr := getenv("ADDR", ":8080")
	log.Printf("server listening on %s", addr)
	log.Printf("swagger UI available at http://%s/swagger/index.html", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("listen: %v", err)
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
