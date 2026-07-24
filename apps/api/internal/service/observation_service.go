// Package service provides business logic orchestration for observation records and species statistics.
package service

import (
	"context"

	"wildpulse/apps/api/internal/repository"
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
}

// Service implements the ObservationService interface.
type Service struct {
	repo repository.ObservationRepository
}

// NewObservationService initializes a new Service instance with the given repository.
func NewObservationService(repo repository.ObservationRepository) *Service {
	return &Service{repo: repo}
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
