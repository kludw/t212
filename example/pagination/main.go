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
	defer c.Close()

	ctx := context.Background()

	count := 0
	for o, err := range c.HistoricalOrdersIter(ctx, nil) {
		if err != nil {
			log.Fatalf("page error after %d orders: %v", count, err)
		}
		count++
		_ = o
	}
	fmt.Printf("walked %d historical orders\n", count)
}
