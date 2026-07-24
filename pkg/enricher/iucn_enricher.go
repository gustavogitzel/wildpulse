package enricher

import (
	"context"
	"log"
	"strings"

	"wildpulse/pkg/domain"
)

type IUCNEnricher struct {
	knownThreats map[string]domain.IUCNStatus
}

func NewIUCNEnricher() *IUCNEnricher {
	return &IUCNEnricher{
		knownThreats: map[string]domain.IUCNStatus{
			"panthera onca":              domain.StatusVU,
			"leontopithecus rosalia":     domain.StatusEN,
			"anodorhynchus hyacinthinus": domain.StatusVU,
			"chrysocyon brachyurus":      domain.StatusNT,
			"trichechus inunguis":        domain.StatusVU,
			"priodontes maximus":         domain.StatusVU,
			"myrmecophaga tridactyla":    domain.StatusVU,
			"tapirus terrestris":         domain.StatusVU,
			"panthera pardus":            domain.StatusVU,
			"leopardus pardalis":         domain.StatusLC,
		},
	}
}

// EnrichObservations updates observation threat statuses based on IUCN Red List taxonomic matching.
func (e *IUCNEnricher) EnrichObservations(ctx context.Context, observations []domain.Observation) int {
	enrichedCount := 0
	for i := range observations {
		name := strings.ToLower(strings.TrimSpace(observations[i].ScientificName))
		if status, found := e.knownThreats[name]; found {
			observations[i].IUCNStatus = status
			enrichedCount++
			continue
		}

		// Heuristic classification fallback for demo/ingestion
		if strings.Contains(name, "panthera") || strings.Contains(name, "leontopithecus") {
			observations[i].IUCNStatus = domain.StatusEN
			enrichedCount++
		} else if strings.Contains(name, "anodorhynchus") || strings.Contains(name, "trichechus") {
			observations[i].IUCNStatus = domain.StatusVU
			enrichedCount++
		}
	}

	log.Printf("🛡️ IUCNEnricher: Enriched %d observations with conservation status.", enrichedCount)
	return enrichedCount
}
