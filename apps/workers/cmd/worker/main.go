package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/robfig/cron/v3"
	"wildpulse/apps/workers/internal/collector"
	"wildpulse/apps/workers/internal/enricher"
)

func main() {
	log.Println("🦫 WildPulse Ingestion Workers starting...")

	gbifCollector := collector.NewGBIFCollector(5)
	iucnEnricher := enricher.NewIUCNEnricher()

	runIngestionJob := func() {
		log.Println("⏰ Executing scheduled GBIF & IUCN sync job...")
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		obs, err := gbifCollector.FetchSouthAmericaOccurrences(ctx, 3)
		if err != nil {
			log.Printf("❌ GBIF ingestion error: %v", err)
			return
		}

		iucnEnricher.EnrichObservations(ctx, obs)
		log.Printf("✅ Ingestion job finished: %d records ingested and ready.", len(obs))
	}

	// Run initial ingestion immediately on startup
	runIngestionJob()

	// Initialize Cron Scheduler (runs every 30 minutes)
	c := cron.New()
	_, err := c.AddFunc("@every 30m", runIngestionJob)
	if err != nil {
		log.Fatalf("❌ Failed to schedule cron job: %v", err)
	}

	c.Start()
	log.Println("📅 Cron scheduler started. Syncing GBIF data every 30 minutes.")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("🛑 Shutting down WildPulse Workers cleanly...")
	c.Stop()
	log.Println("👋 Workers terminated gracefully.")
}
