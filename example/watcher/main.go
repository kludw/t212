package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

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
		OnPositionOpen: func(p *t212.Position) {
			log.Printf("position OPEN  %s qty=%.2f", p.GetTicker(), p.GetQuantity())
		},
		OnPositionClose: func(p *t212.Position) {
			log.Printf("position CLOSE %s qty=%.2f", p.GetTicker(), p.GetQuantity())
		},
		OnPollError: func(err error) {
			log.Printf("poll error: %v", err)
		},
	})
	if err != nil {
		log.Fatalf("could not create client: %v", err)
	}
	defer c.Close()

	log.Printf("watching %d open positions; press Ctrl+C to exit", len(c.Snapshot()))

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
}
