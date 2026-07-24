package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"wildpulse/apps/api/internal/service"
	"wildpulse/pkg/domain"
)

type mockRepo struct{}

func (m *mockRepo) GetObservations(ctx context.Context, filter domain.ObservationFilter) (*domain.PaginatedResult[domain.Observation], error) {
	return &domain.PaginatedResult[domain.Observation]{
		Data: []domain.Observation{
			{
				ID:             1,
				TaxonKey:       2435099,
				SpeciesName:    "Onça-pintada",
				ScientificName: "Panthera onca",
				Latitude:       -3.1190,
				Longitude:      -60.0217,
				Biome:          "Amazônia",
				IUCNStatus:     domain.StatusVU,
				EventDate:      time.Now(),
				CreatedAt:      time.Now(),
			},
		},
		Total:       1,
		Limit:       filter.Limit,
		Offset:      filter.Offset,
		HasNextPage: false,
	}, nil
}

func (m *mockRepo) GetSpeciesByID(ctx context.Context, id int64) (*domain.Species, []domain.Observation, error) {
	sp := &domain.Species{
		ID:             id,
		TaxonKey:       id,
		SpeciesName:    "Onça-pintada",
		ScientificName: "Panthera onca",
		IUCNStatus:     domain.StatusVU,
		TotalCount:     1,
	}
	obs := []domain.Observation{
		{
			ID:             1,
			TaxonKey:       id,
			SpeciesName:    "Onça-pintada",
			ScientificName: "Panthera onca",
			Biome:          "Amazônia",
			IUCNStatus:     domain.StatusVU,
		},
	}
	return sp, obs, nil
}

func (m *mockRepo) GetStatsSummary(ctx context.Context) (*domain.StatsSummary, error) {
	return &domain.StatsSummary{
		TotalObservations: 1420,
		TotalSpecies:      184,
		ThreatenedSpecies: 32,
		ActiveBiomes:      6,
		Biomes: []domain.BiomeStat{
			{Biome: "Amazônia", Count: 500},
		},
		LastIngestTime: time.Now(),
	}, nil
}

func (m *mockRepo) SaveObservations(ctx context.Context, obs []domain.Observation) (int, error) {
	return len(obs), nil
}

type mockService struct {
	*service.Service
	mockRepo *mockRepo
}

func (ms *mockService) TriggerIngestion(ctx context.Context) (int, error) {
	return 10, nil
}

func TestAPIHandlers(t *testing.T) {
	repo := &mockRepo{}
	svc := service.NewObservationService(repo)
	hnd := NewHandler(svc)

	r := chi.NewRouter()
	hnd.RegisterRoutes(r)

	t.Run("GET /health", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK, got %d", rec.Code)
		}
	})

	t.Run("GET /api/v1/observations", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/observations?biome=Amaz%C3%B4nia&limit=5", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK, got %d", rec.Code)
		}

		var res domain.PaginatedResult[domain.Observation]
		if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(res.Data) == 0 {
			t.Errorf("Expected observation results, got empty slice")
		}
	})

	t.Run("POST /api/v1/observations/trigger", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/observations/trigger", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK, got %d", rec.Code)
		}
	})

	t.Run("GET /api/v1/stats", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/stats", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK, got %d", rec.Code)
		}

		var stats domain.StatsSummary
		if err := json.NewDecoder(rec.Body).Decode(&stats); err != nil {
			t.Fatalf("Failed to decode stats: %v", err)
		}

		if stats.TotalObservations == 0 {
			t.Errorf("Expected total observations > 0")
		}
	})
}
