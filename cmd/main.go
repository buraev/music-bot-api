package main

import (
	"database/sql"
	"log"
	"music-bot-api/internal/config"
	"music-bot-api/internal/handler"
	"music-bot-api/internal/middleware"
	"music-bot-api/internal/repository"
	"music-bot-api/internal/service"
	"net/http"

	_ "modernc.org/sqlite"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open("sqlite", cfg.DatabasePath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	tagRepo := repository.NewTagRepo(db)
	linkRepo := repository.NewLinkRepo(db)

	tagService := service.NewTagService(tagRepo)
	linkService := service.NewLinkService(linkRepo)

	tagHandler := handler.NewTagHandler(tagService)
	linkHandler := handler.NewLinkHandler(linkService)

	http.Handle("/tags", middleware.WithCORS(tagHandler))
	http.Handle("/links", middleware.WithCORS(linkHandler))

	log.Println("Server started on:", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, nil))
}
