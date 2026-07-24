package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"wildpulse/pkg/domain"
	"wildpulse/pkg/spatial"
)

type GBIFOccurrenceResult struct {
	Key            int64   `json:"key"`
	SpeciesKey     int64   `json:"speciesKey"`
	TaxonKey       int64   `json:"taxonKey"`
	Species        string  `json:"species"`
	ScientificName string  `json:"scientificName"`
	DecimalLat     float64 `json:"decimalLatitude"`
	DecimalLng     float64 `json:"decimalLongitude"`
	Country        string  `json:"country"`
	Locality       string  `json:"locality"`
	EventDate      string  `json:"eventDate"`
	Media          []struct {
		Type        string `json:"type"`
		Identifier  string `json:"identifier"`
		Format      string `json:"format"`
	} `json:"media"`
}

type GBIFSearchResponse struct {
	Offset     int                    `json:"offset"`
	Limit      int                    `json:"limit"`
	EndOfRecords bool                 `json:"endOfRecords"`
	Count      int                    `json:"count"`
	Results    []GBIFOccurrenceResult `json:"results"`
}

type GBIFCollector struct {
	httpClient *http.Client
	workerCount int
}

func NewGBIFCollector(workerCount int) *GBIFCollector {
	if workerCount <= 0 {
		workerCount = 5
	}
	return &GBIFCollector{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		workerCount: workerCount,
	}
}

// FetchSouthAmericaOccurrences fetches occurrences concurrently using worker pools.
func (c *GBIFCollector) FetchSouthAmericaOccurrences(ctx context.Context, pages int) ([]domain.Observation, error) {
	log.Printf("🌿 GBIFCollector: Starting concurrent fetch across %d pages (%d workers)...", pages, c.workerCount)

	jobs := make(chan int, pages)
	results := make(chan []domain.Observation, pages)
	var wg sync.WaitGroup

	// Launch worker pool
	for w := 1; w <= c.workerCount; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for page := range jobs {
				obs, err := c.fetchPage(ctx, page)
				if err != nil {
					log.Printf("⚠️ Worker %d page %d failed: %v", workerID, page, err)
					results <- nil
					continue
				}
				results <- obs
			}
		}(w)
	}

	// Enqueue jobs
	for p := 0; p < pages; p++ {
		jobs <- p
	}
	close(jobs)

	// Wait for workers in separate goroutine and close results channel
	go func() {
		wg.Wait()
		close(results)
	}()

	var allObservations []domain.Observation
	for pageResult := range results {
		if pageResult != nil {
			allObservations = append(allObservations, pageResult...)
		}
	}

	log.Printf("✨ GBIFCollector: Fetched and processed %d valid occurrences with media.", len(allObservations))
	return allObservations, nil
}

func (c *GBIFCollector) fetchPage(ctx context.Context, page int) ([]domain.Observation, error) {
	limit := 30
	offset := page * limit

	baseURL := "https://api.gbif.org/v1/occurrence/search"
	params := url.Values{}
	params.Add("decimalLatitude", "-34,5")   // South America latitude range
	params.Add("decimalLongitude", "-74,-34") // South America longitude range
	params.Add("mediaType", "StillImage")
	params.Add("hasCoordinate", "true")
	params.Add("limit", strconv.Itoa(limit))
	params.Add("offset", strconv.Itoa(offset))

	reqURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "WildPulse-Worker/1.0 (Go; +https://wildpulse.org)")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GBIF API HTTP %d", resp.StatusCode)
	}

	var searchResp GBIFSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, err
	}

	var observations []domain.Observation
	for _, rec := range searchResp.Results {
		if rec.DecimalLat == 0 || rec.DecimalLng == 0 {
			continue
		}

		imageURL := ""
		for _, m := range rec.Media {
			if m.Identifier != "" {
				imageURL = m.Identifier
				break
			}
		}
		if imageURL == "" {
			continue
		}

		eventTime := time.Now()
		if rec.EventDate != "" {
			if parsed, err := time.Parse(time.RFC3339, rec.EventDate); err == nil {
				eventTime = parsed
			} else if parsed, err := time.Parse("2006-01-02", rec.EventDate); err == nil {
				eventTime = parsed
			}
		}

		speciesName := rec.Species
		if speciesName == "" {
			speciesName = rec.ScientificName
		}
		if speciesName == "" {
			speciesName = "Fauna/Flora Selvagem"
		}

		taxonKey := rec.TaxonKey
		if taxonKey == 0 {
			taxonKey = rec.SpeciesKey
		}
		if taxonKey == 0 {
			taxonKey = rec.Key
		}

		biome := spatial.DetectBiome(rec.DecimalLat, rec.DecimalLng)

		obs := domain.Observation{
			ID:             rec.Key,
			TaxonKey:       taxonKey,
			SpeciesName:    speciesName,
			ScientificName: rec.ScientificName,
			Latitude:       rec.DecimalLat,
			Longitude:      rec.DecimalLng,
			ImageURL:       imageURL,
			EventDate:      eventTime,
			Biome:          biome,
			Country:        rec.Country,
			Locality:       rec.Locality,
			DatasetKey:     "gbif-occurrence-api",
			IUCNStatus:     domain.StatusLC, // Default, enriched by IUCN worker
			CreatedAt:      time.Now(),
		}
		observations = append(observations, obs)
	}

	return observations, nil
}
