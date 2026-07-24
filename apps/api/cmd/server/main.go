package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"

	"wildpulse/apps/api/internal/handler"
	"wildpulse/apps/api/internal/service"
	"wildpulse/pkg/database"
	"wildpulse/pkg/repository"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/wildpulse?sslmode=disable"
	}

	var pool *pgxpool.Pool
	var poolErr error

	// Connect with Retry Loop (tries up to 5 times over 10 seconds for local Docker DB startup)
	for attempt := 1; attempt <= 5; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		p, err := pgxpool.New(ctx, dbURL)
		if err == nil && p.Ping(ctx) == nil {
			pool = p
			cancel()
			log.Println("✅ Successfully connected to PostgreSQL / PostGIS database!")
			break
		}
		if p != nil {
			p.Close()
		}
		poolErr = err
		cancel()
		if attempt < 5 {
			log.Printf("⏳ Waiting for database connection (attempt %d/5)...", attempt)
			time.Sleep(2 * time.Second)
		}
	}

	if pool == nil {
		log.Fatalf("❌ FATAL: Unable to connect to PostgreSQL / PostGIS database (%v). Please ensure Docker is running ('docker compose up -d') or pass DATABASE_URL.", poolErr)
	}

	// Run Database Migrations
	migrationCtx, migrationCancel := context.WithTimeout(context.Background(), 15*time.Second)
	if err := database.RunMigrations(migrationCtx, pool); err != nil {
		log.Printf("⚠️ Migration warning: %v", err)
	}
	migrationCancel()

	// Initialize Layers
	repo := repository.NewPostgresRepository(pool)
	svc := service.NewObservationService(repo)
	hnd := handler.NewHandler(svc)

	r := chi.NewRouter()

	// Middlewares
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	hnd.RegisterRoutes(r)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("🚀 WildPulse REST API server running on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Server failure: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("🛑 Shutting down WildPulse API server gracefully...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Error shutting down server: %v", err)
	}
	if pool != nil {
		pool.Close()
	}
	log.Println("👋 Server stopped cleanly.")
}
