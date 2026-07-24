package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/robfig/cron/v3"
	"wildpulse/pkg/collector"
	"wildpulse/pkg/enricher"
	"wildpulse/pkg/repository"
)

func main() {
	log.Println("🦫 WildPulse Ingestion Workers starting...")

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/wildpulse?sslmode=disable"
	}

	var pool *pgxpool.Pool
	for attempt := 1; attempt <= 3; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		p, err := pgxpool.New(ctx, dbURL)
		if err == nil && p.Ping(ctx) == nil {
			pool = p
			cancel()
			log.Println("✅ Worker successfully connected to PostgreSQL / PostGIS database!")
			break
		}
		if p != nil {
			p.Close()
		}
		cancel()
		time.Sleep(1 * time.Second)
	}

	if pool == nil {
		log.Println("⚠️ Worker warning: Unable to connect to database. Records will be collected in-memory without DB persistence.")
	}

	repo := repository.NewPostgresRepository(pool)
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
		if pool != nil {
			saved, saveErr := repo.SaveObservations(ctx, obs)
			if saveErr != nil {
				log.Printf("⚠️ Error saving observations to database: %v", saveErr)
			} else {
				log.Printf("✅ Ingestion job finished: %d / %d records saved to database.", saved, len(obs))
			}
		} else {
			log.Printf("✅ Ingestion job finished: %d records fetched.", len(obs))
		}
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
	if pool != nil {
		pool.Close()
	}
	log.Println("👋 Workers terminated gracefully.")
}
