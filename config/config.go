package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func init() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found")
	}

	// Verify required environment variables
	required := []string{
		"OPENAI_API_KEY",
	}

	for _, env := range required {
		if os.Getenv(env) == "" {
			log.Printf("Warning: %s environment variable not set\n", env)
		}
	}
}
