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

	_ "github.com/jstowers/pokemon/docs"
	"github.com/jstowers/pokemon/internal/config"
	"github.com/jstowers/pokemon/internal/db"
	"github.com/jstowers/pokemon/internal/handler"
	"github.com/jstowers/pokemon/internal/pokemon"
	"github.com/jstowers/pokemon/internal/repository"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

func main() {
	cfg, err := config.Load(".env.dev")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	database, err := db.Open(db.Config{
		Host:     cfg.DBHost,
		Port:     cfg.DBPort,
		User:     cfg.DBUser,
		Password: cfg.DBPassword,
		DBName:   cfg.DBName,
		SSLMode:  cfg.DBSSLMode,
	})
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

	log.Printf("server listening on %s", cfg.Addr)
	log.Printf("swagger UI available at http://localhost%s/swagger/index.html", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, mux); err != nil {
		log.Fatalf("listen: %v", err)
	}
}
