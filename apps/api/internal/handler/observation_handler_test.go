package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"wildpulse/apps/api/internal/repository"
	"wildpulse/apps/api/internal/service"
	"wildpulse/pkg/domain"
)

func TestAPIHandlers(t *testing.T) {
	repo := repository.NewPostgresRepository(nil)
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
