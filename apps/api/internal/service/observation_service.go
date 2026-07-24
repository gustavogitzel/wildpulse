// Package service provides business logic orchestration for observation records and species statistics.
package service

import (
	"context"
	"log"
	"time"

	"wildpulse/pkg/repository"

	"wildpulse/pkg/collector"
	"wildpulse/pkg/enricher"

	"wildpulse/pkg/domain"
)

// ObservationService defines high-level business operations for API endpoints.
type ObservationService interface {
	// GetObservations retrieves paginated observation records based on filtering criteria.
	GetObservations(ctx context.Context, filter domain.ObservationFilter) (*domain.PaginatedResult[domain.Observation], error)

	// GetSpeciesByID fetches taxonomy metadata and occurrences for a given species ID.
	GetSpeciesByID(ctx context.Context, id int64) (*domain.SpeciesDetailsResponse, error)

	// GetStatsSummary aggregates platform-wide metrics, biome counts, and threat stats.
	GetStatsSummary(ctx context.Context) (*domain.StatsSummary, error)

	// TriggerIngestion executes on-demand GBIF collection & IUCN enrichment, saving results to the DB.
	TriggerIngestion(ctx context.Context) (int, error)
}

// Service implements the ObservationService interface.
type Service struct {
	repo          repository.ObservationRepository
	gbifCollector *collector.GBIFCollector
	iucnEnricher  *enricher.IUCNEnricher
}

// NewObservationService initializes a new Service instance with the given repository.
func NewObservationService(repo repository.ObservationRepository) *Service {
	return &Service{
		repo:          repo,
		gbifCollector: collector.NewGBIFCollector(5),
		iucnEnricher:  enricher.NewIUCNEnricher(),
	}
}

// GetObservations delegates observation queries to the repository layer.
func (s *Service) GetObservations(ctx context.Context, filter domain.ObservationFilter) (*domain.PaginatedResult[domain.Observation], error) {
	return s.repo.GetObservations(ctx, filter)
}

// GetSpeciesByID fetches species info and historical occurrences.
func (s *Service) GetSpeciesByID(ctx context.Context, id int64) (*domain.SpeciesDetailsResponse, error) {
	sp, occurrences, err := s.repo.GetSpeciesByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return &domain.SpeciesDetailsResponse{
		Species:     *sp,
		Occurrences: occurrences,
	}, nil
}

// GetStatsSummary returns aggregated platform metrics.
func (s *Service) GetStatsSummary(ctx context.Context) (*domain.StatsSummary, error) {
	return s.repo.GetStatsSummary(ctx)
}

// TriggerIngestion triggers on-demand GBIF + IUCN ingestion and saves observations to DB.
func (s *Service) TriggerIngestion(ctx context.Context) (int, error) {
	log.Println("⚡ Triggering on-demand GBIF & IUCN ingestion pipeline...")
	timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	obs, err := s.gbifCollector.FetchSouthAmericaOccurrences(timeoutCtx, 3)
	if err != nil {
		log.Printf("❌ On-demand GBIF collection error: %v", err)
		return 0, err
	}

	s.iucnEnricher.EnrichObservations(timeoutCtx, obs)
	savedCount, err := s.repo.SaveObservations(timeoutCtx, obs)
	if err != nil {
		log.Printf("❌ Failed to save ingested observations: %v", err)
		return 0, err
	}

	log.Printf("✅ Triggered ingestion pipeline completed: %d records saved.", savedCount)
	return savedCount, nil
}
