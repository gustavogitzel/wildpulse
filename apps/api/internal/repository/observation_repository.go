package repository

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"wildpulse/pkg/domain"
	"wildpulse/pkg/spatial"
)

type ObservationRepository interface {
	GetObservations(ctx context.Context, filter domain.ObservationFilter) (*domain.PaginatedResult[domain.Observation], error)
	GetSpeciesByID(ctx context.Context, id int64) (*domain.Species, []domain.Observation, error)
	GetStatsSummary(ctx context.Context) (*domain.StatsSummary, error)
	SaveObservations(ctx context.Context, obs []domain.Observation) (int, error)
}

type PostgresRepository struct {
	pool *pgxpool.Pool
	mu   sync.RWMutex
	// In-memory fallback database for dev mode or when db pool is nil
	mockObservations map[int64]domain.Observation
	mockSpecies      map[int64]domain.Species
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	repo := &PostgresRepository{
		pool:             pool,
		mockObservations: make(map[int64]domain.Observation),
		mockSpecies:      make(map[int64]domain.Species),
	}
	repo.initMockData()
	return repo
}

func (r *PostgresRepository) GetObservations(ctx context.Context, filter domain.ObservationFilter) (*domain.PaginatedResult[domain.Observation], error) {
	if r.pool != nil {
		return r.getObservationsFromDB(ctx, filter)
	}
	return r.getObservationsFromMemory(filter), nil
}

func (r *PostgresRepository) GetSpeciesByID(ctx context.Context, id int64) (*domain.Species, []domain.Observation, error) {
	if r.pool != nil {
		sp, obs, err := r.getSpeciesFromDB(ctx, id)
		if err == nil && sp != nil {
			return sp, obs, nil
		}
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	sp, exists := r.mockSpecies[id]
	if !exists {
		// Fallback lookup by TaxonKey if ID search is passed
		for _, s := range r.mockSpecies {
			if s.TaxonKey == id {
				sp = s
				exists = true
				break
			}
		}
	}
	if !exists {
		return nil, nil, fmt.Errorf("species with ID %d not found", id)
	}

	var occurrences []domain.Observation
	for _, o := range r.mockObservations {
		if o.TaxonKey == sp.TaxonKey {
			occurrences = append(occurrences, o)
		}
	}
	return &sp, occurrences, nil
}

func (r *PostgresRepository) GetStatsSummary(ctx context.Context) (*domain.StatsSummary, error) {
	if r.pool != nil {
		stats, err := r.getStatsFromDB(ctx)
		if err == nil {
			return stats, nil
		}
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	var threatened int64
	for _, sp := range r.mockSpecies {
		if sp.IUCNStatus.IsThreatened() {
			threatened++
		}
	}

	biomeCounts := make(map[string]int64)
	for _, obs := range r.mockObservations {
		biomeCounts[obs.Biome]++
	}

	var biomeStats []domain.BiomeStat
	for b, count := range biomeCounts {
		biomeStats = append(biomeStats, domain.BiomeStat{Biome: b, Count: count})
	}

	return &domain.StatsSummary{
		TotalObservations: int64(len(r.mockObservations)),
		TotalSpecies:      int64(len(r.mockSpecies)),
		ThreatenedSpecies: threatened,
		ActiveBiomes:      len(biomeCounts),
		Biomes:            biomeStats,
		LastIngestTime:    time.Now().Add(-12 * time.Minute),
	}, nil
}

func (r *PostgresRepository) SaveObservations(ctx context.Context, obs []domain.Observation) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	saved := 0
	for _, o := range obs {
		if o.ID == 0 {
			o.ID = int64(len(r.mockObservations) + 1001)
		}
		if o.CreatedAt.IsZero() {
			o.CreatedAt = time.Now()
		}
		r.mockObservations[o.ID] = o
		saved++

		// Upsert species record
		if _, exists := r.mockSpecies[o.TaxonKey]; !exists {
			r.mockSpecies[o.TaxonKey] = domain.Species{
				ID:             o.TaxonKey,
				TaxonKey:       o.TaxonKey,
				SpeciesName:    o.SpeciesName,
				ScientificName: o.ScientificName,
				IUCNStatus:     o.IUCNStatus,
				ImageURL:       o.ImageURL,
				TotalCount:     1,
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			}
		} else {
			sp := r.mockSpecies[o.TaxonKey]
			sp.TotalCount++
			sp.UpdatedAt = time.Now()
			r.mockSpecies[o.TaxonKey] = sp
		}
	}
	return saved, nil
}

func (r *PostgresRepository) getObservationsFromDB(ctx context.Context, filter domain.ObservationFilter) (*domain.PaginatedResult[domain.Observation], error) {
	query := `SELECT id, taxon_key, species_name, scientific_name, latitude, longitude, image_url, event_date, biome, country, locality, dataset_key, iucn_status, created_at FROM observations WHERE 1=1`
	var args []any
	argIdx := 1

	if filter.Biome != "" {
		query += fmt.Sprintf(" AND biome = $%d", argIdx)
		args = append(args, filter.Biome)
		argIdx++
	}

	if filter.MinLat != nil && filter.MaxLat != nil && filter.MinLng != nil && filter.MaxLng != nil {
		bbox := spatial.BBox{
			MinLat: *filter.MinLat, MaxLat: *filter.MaxLat,
			MinLng: *filter.MinLng, MaxLng: *filter.MaxLng,
		}
		envelope := spatial.BuildPostGISEnvelopeSQL(bbox, 4326)
		query += fmt.Sprintf(" AND ST_Within(location_geom, %s)", envelope)
	}

	if filter.Taxa != "" {
		query += fmt.Sprintf(" AND (species_name ILIKE $%d OR scientific_name ILIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+filter.Taxa+"%")
		argIdx++
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	query += fmt.Sprintf(" ORDER BY event_date DESC LIMIT %d OFFSET %d", limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.Observation
	for rows.Next() {
		var o domain.Observation
		if err := rows.Scan(&o.ID, &o.TaxonKey, &o.SpeciesName, &o.ScientificName, &o.Latitude, &o.Longitude, &o.ImageURL, &o.EventDate, &o.Biome, &o.Country, &o.Locality, &o.DatasetKey, &o.IUCNStatus, &o.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, o)
	}

	return &domain.PaginatedResult[domain.Observation]{
		Data:        list,
		Total:       int64(len(list)),
		Limit:       limit,
		Offset:      offset,
		HasNextPage: len(list) == limit,
	}, nil
}

func (r *PostgresRepository) getSpeciesFromDB(ctx context.Context, id int64) (*domain.Species, []domain.Observation, error) {
	var sp domain.Species
	querySp := `SELECT id, taxon_key, species_name, scientific_name, kingdom, phylum, class, order_name, family, iucn_status, description, image_url, total_count, created_at, updated_at FROM species WHERE id = $1`
	err := r.pool.QueryRow(ctx, querySp, id).Scan(&sp.ID, &sp.TaxonKey, &sp.SpeciesName, &sp.ScientificName, &sp.Kingdom, &sp.Phylum, &sp.Class, &sp.Order, &sp.Family, &sp.IUCNStatus, &sp.Description, &sp.ImageURL, &sp.TotalCount, &sp.CreatedAt, &sp.UpdatedAt)
	if err != nil {
		return nil, nil, err
	}

	queryObs := `SELECT id, taxon_key, species_name, scientific_name, latitude, longitude, image_url, event_date, biome, country, locality, dataset_key, iucn_status, created_at FROM observations WHERE taxon_key = $1 ORDER BY event_date DESC LIMIT 50`
	rows, err := r.pool.Query(ctx, queryObs, sp.TaxonKey)
	if err != nil {
		return &sp, nil, nil
	}
	defer rows.Close()

	var obsList []domain.Observation
	for rows.Next() {
		var o domain.Observation
		if err := rows.Scan(&o.ID, &o.TaxonKey, &o.SpeciesName, &o.ScientificName, &o.Latitude, &o.Longitude, &o.ImageURL, &o.EventDate, &o.Biome, &o.Country, &o.Locality, &o.DatasetKey, &o.IUCNStatus, &o.CreatedAt); err != nil {
			break
		}
		obsList = append(obsList, o)
	}

	return &sp, obsList, nil
}

func (r *PostgresRepository) getStatsFromDB(ctx context.Context) (*domain.StatsSummary, error) {
	var stats domain.StatsSummary
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM observations`).Scan(&stats.TotalObservations)
	if err != nil {
		return nil, err
	}
	_ = r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM species`).Scan(&stats.TotalSpecies)
	_ = r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM species WHERE iucn_status IN ('CR', 'EN', 'VU')`).Scan(&stats.ThreatenedSpecies)

	rows, err := r.pool.Query(ctx, `SELECT biome, COUNT(*) FROM observations GROUP BY biome`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var b domain.BiomeStat
			if err := rows.Scan(&b.Biome, &b.Count); err == nil {
				stats.Biomes = append(stats.Biomes, b)
			}
		}
	}
	stats.ActiveBiomes = len(stats.Biomes)
	stats.LastIngestTime = time.Now()
	return &stats, nil
}

func (r *PostgresRepository) getObservationsFromMemory(filter domain.ObservationFilter) *domain.PaginatedResult[domain.Observation] {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var filtered []domain.Observation
	for _, o := range r.mockObservations {
		if filter.Biome != "" && !strings.EqualFold(o.Biome, filter.Biome) {
			continue
		}
		if filter.Taxa != "" {
			query := strings.ToLower(filter.Taxa)
			if !strings.Contains(strings.ToLower(o.SpeciesName), query) && !strings.Contains(strings.ToLower(o.ScientificName), query) {
				continue
			}
		}
		if filter.MinLat != nil && filter.MaxLat != nil && filter.MinLng != nil && filter.MaxLng != nil {
			bbox := spatial.BBox{MinLat: *filter.MinLat, MaxLat: *filter.MaxLat, MinLng: *filter.MinLng, MaxLng: *filter.MaxLng}
			if !bbox.Contains(o.Latitude, o.Longitude) {
				continue
			}
		}
		if filter.Status != "" && o.IUCNStatus != filter.Status {
			continue
		}
		filtered = append(filtered, o)
	}

	total := int64(len(filtered))
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	start := offset
	if start > len(filtered) {
		start = len(filtered)
	}
	end := start + limit
	if end > len(filtered) {
		end = len(filtered)
	}

	slice := filtered[start:end]
	return &domain.PaginatedResult[domain.Observation]{
		Data:        slice,
		Total:       total,
		Limit:       limit,
		Offset:      offset,
		HasNextPage: end < len(filtered),
	}
}

func (r *PostgresRepository) initMockData() {
	sampleSpecies := []domain.Species{
		{ID: 2435099, TaxonKey: 2435099, SpeciesName: "Onça-pintada", ScientificName: "Panthera onca", Kingdom: "Animalia", Class: "Mammalia", Order: "Carnivora", Family: "Felidae", IUCNStatus: domain.StatusVU, ImageURL: "https://images.unsplash.com/photo-1541781774459-bb2af2f05b55", TotalCount: 142},
		{ID: 5219426, TaxonKey: 5219426, SpeciesName: "Mico-leão-dourado", ScientificName: "Leontopithecus rosalia", Kingdom: "Animalia", Class: "Mammalia", Order: "Primates", Family: "Callitrichidae", IUCNStatus: domain.StatusEN, ImageURL: "https://images.unsplash.com/photo-1574063413132-355dbfd83e0c", TotalCount: 88},
		{ID: 2435240, TaxonKey: 2435240, SpeciesName: "Arara-azul-grande", ScientificName: "Anodorhynchus hyacinthinus", Kingdom: "Animalia", Class: "Aves", Order: "Psittaciformes", Family: "Psittacidae", IUCNStatus: domain.StatusVU, ImageURL: "https://images.unsplash.com/photo-1552728089-57bdde30beb3", TotalCount: 215},
		{ID: 2436444, TaxonKey: 2436444, SpeciesName: "Lobo-guará", ScientificName: "Chrysocyon brachyurus", Kingdom: "Animalia", Class: "Mammalia", Order: "Carnivora", Family: "Canidae", IUCNStatus: domain.StatusNT, ImageURL: "https://images.unsplash.com/photo-1564349683136-77e08dba1ef9", TotalCount: 96},
		{ID: 2440938, TaxonKey: 2440938, SpeciesName: "Peixe-boi-da-Amazônia", ScientificName: "Trichechus inunguis", Kingdom: "Animalia", Class: "Mammalia", Order: "Sirenia", Family: "Trichechidae", IUCNStatus: domain.StatusVU, ImageURL: "https://images.unsplash.com/photo-1582967788606-a171c1080cb0", TotalCount: 45},
	}

	for _, s := range sampleSpecies {
		r.mockSpecies[s.ID] = s
	}

	biomes := []string{"Amazônia", "Cerrado", "Mata Atlântica", "Pantanal", "Caatinga", "Pampa"}
	locations := map[string][][2]float64{
		"Amazônia":       {{-3.1190, -60.0217}, {-1.4558, -48.4902}, {-9.9754, -67.8249}},
		"Cerrado":        {{-15.7939, -47.8828}, {-16.6869, -49.2648}, {-10.1844, -48.3336}},
		"Mata Atlântica": {{-23.5505, -46.6333}, {-22.9068, -43.1729}, {-25.4284, -49.2733}},
		"Pantanal":       {{-19.0089, -57.6534}, {-16.2750, -56.6211}},
		"Caatinga":       {{-9.4069, -40.5028}, {-7.1153, -34.8610}},
		"Pampa":          {{-30.0346, -51.2177}, {-31.7654, -52.3376}},
	}

	idCounter := int64(1)
	for i := 0; i < 40; i++ {
		sp := sampleSpecies[rand.Intn(len(sampleSpecies))]
		biome := biomes[rand.Intn(len(biomes))]
		coords := locations[biome][rand.Intn(len(locations[biome]))]

		obs := domain.Observation{
			ID:             idCounter,
			TaxonKey:       sp.TaxonKey,
			SpeciesName:    sp.SpeciesName,
			ScientificName: sp.ScientificName,
			Latitude:       coords[0] + (rand.Float64()-0.5)*0.5,
			Longitude:      coords[1] + (rand.Float64()-0.5)*0.5,
			ImageURL:       sp.ImageURL,
			EventDate:      time.Now().AddDate(0, 0, -rand.Intn(180)),
			Biome:          biome,
			Country:        "Brasil",
			Locality:       fmt.Sprintf("Reserva %s #%d", biome, i+1),
			DatasetKey:     "gbif-south-america-2026",
			IUCNStatus:     sp.IUCNStatus,
			CreatedAt:      time.Now(),
		}
		r.mockObservations[obs.ID] = obs
		idCounter++
	}
}
