package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port         string
	DatabasePath string
}

func Load() (*Config, error) {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found or could not be loaded")
	}

	dbPath := os.Getenv("DATABASE_PATH")
	port := os.Getenv("PORT")
	if dbPath == "" {
		dbPath = "botdata.db"
	}

	if port == "" {
		log.Println("Warning: PORT not found or could not be")
	}

	return &Config{
		DatabasePath: dbPath,
		Port:         port,
	}, nil
}
