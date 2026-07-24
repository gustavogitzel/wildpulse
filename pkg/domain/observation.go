package domain

import "time"

// IUCNStatus represents the conservation status according to the IUCN Red List.
type IUCNStatus string

const (
	StatusCR IUCNStatus = "CR" // Critically Endangered
	StatusEN IUCNStatus = "EN" // Endangered
	StatusVU IUCNStatus = "VU" // Vulnerable
	StatusNT IUCNStatus = "NT" // Near Threatened
	StatusLC IUCNStatus = "LC" // Least Concern
	StatusDD IUCNStatus = "DD" // Data Deficient
	StatusNE IUCNStatus = "NE" // Not Evaluated
)

// IsThreatened returns true if the status is Vulnerable, Endangered, or Critically Endangered.
func (s IUCNStatus) IsThreatened() bool {
	return s == StatusCR || s == StatusEN || s == StatusVU
}

// Species represents taxonomic species information.
type Species struct {
	ID             int64      `json:"id"`
	TaxonKey       int64      `json:"taxon_key"`
	SpeciesName    string     `json:"species_name"`
	ScientificName string     `json:"scientific_name"`
	Kingdom        string     `json:"kingdom,omitempty"`
	Phylum         string     `json:"phylum,omitempty"`
	Class          string     `json:"class,omitempty"`
	Order          string     `json:"order,omitempty"`
	Family         string     `json:"family,omitempty"`
	IUCNStatus     IUCNStatus `json:"iucn_status"`
	Description    string     `json:"description,omitempty"`
	ImageURL       string     `json:"image_url,omitempty"`
	TotalCount     int64      `json:"total_count"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// SpeciesDetailsResponse holds species metadata along with its historical occurrences.
type SpeciesDetailsResponse struct {
	Species     Species       `json:"species"`
	Occurrences []Observation `json:"occurrences"`
}


// Observation represents a geospatial biodiversity occurrence record.
type Observation struct {
	ID             int64      `json:"id"`
	TaxonKey       int64      `json:"taxon_key"`
	SpeciesName    string     `json:"species_name"`
	ScientificName string     `json:"scientific_name"`
	Latitude       float64    `json:"latitude"`
	Longitude      float64    `json:"longitude"`
	ImageURL       string     `json:"image_url"`
	EventDate      time.Time  `json:"event_date"`
	Biome          string     `json:"biome"`
	Country        string     `json:"country,omitempty"`
	Locality       string     `json:"locality,omitempty"`
	DatasetKey     string     `json:"dataset_key,omitempty"`
	IUCNStatus     IUCNStatus `json:"iucn_status"`
	CreatedAt      time.Time  `json:"created_at"`
}

// ObservationFilter defines query parameters for fetching observations.
type ObservationFilter struct {
	MinLat   *float64   `json:"min_lat,omitempty"`
	MaxLat   *float64   `json:"max_lat,omitempty"`
	MinLng   *float64   `json:"min_lng,omitempty"`
	MaxLng   *float64   `json:"max_lng,omitempty"`
	Biome    string     `json:"biome,omitempty"`
	Taxa     string     `json:"taxa,omitempty"`
	Status   IUCNStatus `json:"status,omitempty"`
	TaxonKey *int64     `json:"taxon_key,omitempty"`
	Limit    int        `json:"limit"`
	Offset   int        `json:"offset"`
}

// PaginatedResult is a generic wrapper for paginated API responses.
type PaginatedResult[T any] struct {
	Data       []T   `json:"data"`
	Total      int64 `json:"total"`
	Limit      int   `json:"limit"`
	Offset     int   `json:"offset"`
	HasNextPage bool  `json:"has_next_page"`
}

// BiomeStat holds observation counts grouped by biome.
type BiomeStat struct {
	Biome string `json:"biome"`
	Count int64  `json:"count"`
}

// StatsSummary provides aggregated platform metrics for the frontend cards.
type StatsSummary struct {
	TotalObservations int64       `json:"total_observations"`
	TotalSpecies      int64       `json:"total_species"`
	ThreatenedSpecies int64       `json:"threatened_species"`
	ActiveBiomes      int         `json:"active_biomes"`
	Biomes            []BiomeStat `json:"biomes"`
	LastIngestTime    time.Time   `json:"last_ingest_time"`
}
