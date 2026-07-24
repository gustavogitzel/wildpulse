package handler

import (
	_ "embed"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"wildpulse/apps/api/internal/service"
	"wildpulse/pkg/domain"
)

//go:embed docs/swagger.json
var swaggerJSON []byte

type Handler struct {
	svc service.ObservationService
}

func NewHandler(svc service.ObservationService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/observations", h.GetObservations)
		r.Post("/observations/trigger", h.TriggerIngestion)
		r.Get("/species/{id}", h.GetSpeciesByID)
		r.Get("/stats", h.GetStats)
	})
	r.Get("/health", h.Health)
	r.Get("/swagger/doc.json", h.SwaggerJSON)
	r.Get("/swagger*", h.SwaggerUI)
	r.Get("/docs", h.SwaggerUI)
}

func (h *Handler) SwaggerJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(swaggerJSON)
}

func (h *Handler) SwaggerUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	html := `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>WildPulse API - Documentation</title>
  <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
  <style>
    html { box-sizing: border-box; overflow: -moz-scrollbars-vertical; overflow-y: scroll; }
    *, *:before, *:after { box-sizing: inherit; }
    body { margin:0; background: #fafafa; }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js" charset="UTF-8"></script>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-standalone-preset.js" charset="UTF-8"></script>
  <script>
    window.onload = function() {
      const ui = SwaggerUIBundle({
        url: "/swagger/doc.json",
        dom_id: '#swagger-ui',
        deepLinking: true,
        presets: [
          SwaggerUIBundle.presets.apis,
          SwaggerUIStandalonePreset
        ],
        plugins: [
          SwaggerUIBundle.plugins.DownloadUrl
        ],
        layout: "StandaloneLayout"
      });
      window.ui = ui;
    };
  </script>
</body>
</html>`
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(html))
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{
		"status":  "healthy",
		"service": "wildpulse-api",
		"version": "1.0.0",
	})
}

func (h *Handler) GetObservations(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	filter := domain.ObservationFilter{
		Biome:  q.Get("biome"),
		Taxa:   q.Get("taxa"),
		Status: domain.IUCNStatus(q.Get("status")),
	}

	if minLatStr := q.Get("min_lat"); minLatStr != "" {
		if v, err := strconv.ParseFloat(minLatStr, 64); err == nil {
			filter.MinLat = &v
		}
	}
	if maxLatStr := q.Get("max_lat"); maxLatStr != "" {
		if v, err := strconv.ParseFloat(maxLatStr, 64); err == nil {
			filter.MaxLat = &v
		}
	}
	if minLngStr := q.Get("min_lng"); minLngStr != "" {
		if v, err := strconv.ParseFloat(minLngStr, 64); err == nil {
			filter.MinLng = &v
		}
	}
	if maxLngStr := q.Get("max_lng"); maxLngStr != "" {
		if v, err := strconv.ParseFloat(maxLngStr, 64); err == nil {
			filter.MaxLng = &v
		}
	}

	if limitStr := q.Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			filter.Limit = l
		}
	}
	if offsetStr := q.Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil {
			filter.Offset = o
		}
	}

	result, err := h.svc.GetObservations(r.Context(), filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to retrieve observations", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, result)
}

func (h *Handler) TriggerIngestion(w http.ResponseWriter, r *http.Request) {
	count, err := h.svc.TriggerIngestion(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to trigger ingestion pipeline", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"status":           "success",
		"message":          "GBIF & IUCN observation ingestion triggered successfully",
		"records_ingested": count,
	})
}

func (h *Handler) GetSpeciesByID(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid species ID format", err.Error())
		return
	}

	details, err := h.svc.GetSpeciesByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Species not found", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, details)
}

func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.svc.GetStatsSummary(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to retrieve statistics", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, stats)
}

func respondJSON(w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}

func respondError(w http.ResponseWriter, code int, message string, details string) {
	respondJSON(w, code, map[string]string{
		"error":   message,
		"details": details,
	})
}
