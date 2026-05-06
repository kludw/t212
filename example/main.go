package main

import (
	"context"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/kludw/t212"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("could not load .env: %v", err)
	}

	c, err := t212.NewClient(&t212.ClientOpts{
		Env:       os.Getenv("ENV"),
		APIKeyID:  os.Getenv("API_KEY_ID"),
		APISecret: os.Getenv("API_SECRET_KEY"),
	})
	if err != nil {
		log.Fatalf("could not create client: %v", err)
	}

	_, err = c.AccountSummary(context.Background())
	if err != nil {
		log.Fatalf("could not get account summary: %v", err)
	}

	_, err = c.Instruments(context.Background())
	if err != nil {
		log.Fatalf("could not get instruments: %v", err)
	}

	_, err = c.Exchanges(context.Background())
	if err != nil {
		log.Fatalf("could not get exchanges: %v", err)
	}
}
