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

	summary, err := c.AccountSummary(context.Background())
	if err != nil {
		return fmt.Errorf("account summary: %w", err)
	}

	fmt.Printf("account %d (%s): total value %.2f, %d positions known\n",
		summary.GetID(),
		summary.GetCurrency(),
		summary.GetTotalValue(),
		len(c.Snapshot()),
	)
	return nil
}
