package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

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

	ctx := context.Background()

	to := time.Now().UTC()
	from := to.AddDate(0, -1, 0)
	enq, err := c.RequestReport(ctx, &t212.PublicReportRequest{
		TimeFrom: &from,
		TimeTo:   &to,
		DataIncluded: &t212.ReportDataIncluded{
			IncludeOrders:       boolPtr(true),
			IncludeTransactions: boolPtr(true),
		},
	})
	if err != nil {
		return fmt.Errorf("enqueuing report: %w", err)
	}
	if enq.ReportId == nil {
		return errors.New("report id missing in response")
	}

	report, err := c.WaitForReport(ctx, *enq.ReportId, &t212.WaitForReportOpts{
		PollInterval: 30 * time.Second,
		MaxWait:      10 * time.Minute,
	})
	if err != nil {
		return fmt.Errorf("waiting for report: %w", err)
	}
	fmt.Println("download:", report.GetDownloadLink())
	return nil
}

func boolPtr(b bool) *bool { return &b }
