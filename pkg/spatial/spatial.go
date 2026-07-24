package spatial

import (
	"fmt"
	"math"
)

const EarthRadiusKm = 6371.0

// BBox represents a geographic bounding box defined by minimum and maximum coordinates.
type BBox struct {
	MinLat float64 `json:"min_lat"`
	MaxLat float64 `json:"max_lat"`
	MinLng float64 `json:"min_lng"`
	MaxLng float64 `json:"max_lng"`
}

// IsValid checks if bounding box coordinates form a logical rectangle.
func (b BBox) IsValid() bool {
	if b.MinLat < -90 || b.MaxLat > 90 || b.MinLng < -180 || b.MaxLng > 180 {
		return false
	}
	return b.MinLat <= b.MaxLat && b.MinLng <= b.MaxLng
}

// Contains returns true if the coordinate point falls inside the bounding box.
func (b BBox) Contains(lat, lng float64) bool {
	return lat >= b.MinLat && lat <= b.MaxLat && lng >= b.MinLng && lng <= b.MaxLng
}

// HaversineDistance calculates the great-circle distance between two points in kilometers.
func HaversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	dLat := degreesToRadians(lat2 - lat1)
	dLon := degreesToRadians(lon2 - lon1)

	rLat1 := degreesToRadians(lat1)
	rLat2 := degreesToRadians(lat2)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Sin(dLon/2)*math.Sin(dLon/2)*math.Cos(rLat1)*math.Cos(rLat2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return EarthRadiusKm * c
}

func degreesToRadians(deg float64) float64 {
	return deg * math.Pi / 180
}

// DetectBiome determines the biome region in South America / Brazil based on lat/lng approximations.
func DetectBiome(lat, lng float64) string {
	// South America / Brazil approximate bounding boxes for biomes
	switch {
	case lat >= -13.0 && lat <= 5.0 && lng >= -74.0 && lng <= -44.0:
		return "Amazônia"
	case lat >= -24.0 && lat <= -2.0 && lng >= -60.0 && lng <= -40.0:
		if lat >= -18.0 && lat <= -3.0 && lng >= -45.0 && lng <= -35.0 {
			return "Caatinga"
		}
		return "Cerrado"
	case lat >= -30.0 && lat <= -6.0 && lng >= -55.0 && lng <= -34.0:
		return "Mata Atlântica"
	case lat >= -22.0 && lat <= -15.0 && lng >= -58.0 && lng <= -54.0:
		return "Pantanal"
	case lat >= -34.0 && lat <= -27.0 && lng >= -57.0 && lng <= -49.0:
		return "Pampa"
	default:
		return "Neotropical"
	}
}

// BuildPostGISEnvelopeSQL generates a PostGIS ST_MakeEnvelope SQL filter string.
func BuildPostGISEnvelopeSQL(b BBox, srid int) string {
	return fmt.Sprintf("ST_MakeEnvelope(%f, %f, %f, %f, %d)", b.MinLng, b.MinLat, b.MaxLng, b.MaxLat, srid)
}
