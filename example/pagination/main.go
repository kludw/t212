package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/kludw/t212"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	if err := godotenv.Load(); err != nil {
		return fmt.Errorf("loading .env: %w", err)
	}

	c, err := t212.NewClient(&t212.ClientOpts{
		Env:       os.Getenv("ENV"),
		APIKeyID:  os.Getenv("API_KEY_ID"),
		APISecret: os.Getenv("API_SECRET_KEY"),
	})
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}
	defer c.Close()

	count := 0
	for o, err := range c.HistoricalOrdersIter(context.Background(), nil) {
		if err != nil {
			return fmt.Errorf("page error after %d orders: %w", count, err)
		}
		count++
		_ = o
	}
	fmt.Printf("walked %d historical orders\n", count)
	return nil
}
