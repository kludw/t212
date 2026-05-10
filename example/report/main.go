package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

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
		log.Fatalf("could not enqueue report: %v", err)
	}
	if enq.ReportId == nil {
		log.Fatal("report id missing in response")
	}

	report, err := c.WaitForReport(ctx, *enq.ReportId, &t212.WaitForReportOpts{
		PollInterval: 30 * time.Second,
		MaxWait:      10 * time.Minute,
	})
	if err != nil {
		log.Fatalf("waiting for report: %v", err)
	}
	fmt.Println("download:", report.GetDownloadLink())
}

func boolPtr(b bool) *bool { return &b }
