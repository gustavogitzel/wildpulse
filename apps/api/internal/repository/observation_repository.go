package repository

import (
	"context"
	"fmt"
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
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{
		pool: pool,
	}
}

func (r *PostgresRepository) GetObservations(ctx context.Context, filter domain.ObservationFilter) (*domain.PaginatedResult[domain.Observation], error) {
	if r.pool == nil {
		return nil, fmt.Errorf("database connection pool is nil")
	}
	return r.getObservationsFromDB(ctx, filter)
}

func (r *PostgresRepository) GetSpeciesByID(ctx context.Context, id int64) (*domain.Species, []domain.Observation, error) {
	if r.pool == nil {
		return nil, nil, fmt.Errorf("database connection pool is nil")
	}
	return r.getSpeciesFromDB(ctx, id)
}

func (r *PostgresRepository) GetStatsSummary(ctx context.Context) (*domain.StatsSummary, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("database connection pool is nil")
	}
	return r.getStatsFromDB(ctx)
}

func (r *PostgresRepository) SaveObservations(ctx context.Context, obs []domain.Observation) (int, error) {
	if r.pool == nil {
		return 0, fmt.Errorf("database connection pool is nil")
	}

	saved := 0
	for _, o := range obs {
		// Upsert species record
		querySpecies := `
			INSERT INTO species (taxon_key, species_name, scientific_name, iucn_status, image_url, total_count, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, 1, NOW(), NOW())
			ON CONFLICT (taxon_key) 
			DO UPDATE SET 
				total_count = species.total_count + 1,
				image_url = EXCLUDED.image_url,
				updated_at = NOW()
		`
		_, _ = r.pool.Exec(ctx, querySpecies, o.TaxonKey, o.SpeciesName, o.ScientificName, string(o.IUCNStatus), o.ImageURL)

		// Insert observation record
		queryObs := `
			INSERT INTO observations (taxon_key, species_name, scientific_name, latitude, longitude, image_url, event_date, biome, country, locality, dataset_key, iucn_status, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW())
		`
		_, err := r.pool.Exec(ctx, queryObs, o.TaxonKey, o.SpeciesName, o.ScientificName, o.Latitude, o.Longitude, o.ImageURL, o.EventDate, o.Biome, o.Country, o.Locality, o.DatasetKey, string(o.IUCNStatus))
		if err == nil {
			saved++
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
		limit = 50
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
	querySp := `SELECT id, taxon_key, species_name, scientific_name, kingdom, phylum, class, order_name, family, iucn_status, description, image_url, total_count, created_at, updated_at FROM species WHERE id = $1 OR taxon_key = $1`
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
