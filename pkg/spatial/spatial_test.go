package spatial

import (
	"testing"
)

func TestHaversineDistance(t *testing.T) {
	// Distance between SP (-23.5505, -46.6333) and Rio (-22.9068, -43.1729) is ~357 km
	dist := HaversineDistance(-23.5505, -46.6333, -22.9068, -43.1729)
	if dist < 350 || dist > 370 {
		t.Errorf("Expected distance ~357km, got %f", dist)
	}
}

func TestBBox(t *testing.T) {
	bbox := BBox{MinLat: -30, MaxLat: 5, MinLng: -75, MaxLng: -34}
	if !bbox.IsValid() {
		t.Errorf("Expected bbox to be valid")
	}

	if !bbox.Contains(-15.79, -47.88) { // Brasilia
		t.Errorf("BBox should contain Brasilia coordinates")
	}

	if bbox.Contains(40.71, -74.00) { // NYC
		t.Errorf("BBox should NOT contain NYC coordinates")
	}
}

func TestDetectBiome(t *testing.T) {
	biomeAmazon := DetectBiome(-3.1190, -60.0217) // Manaus
	if biomeAmazon != "Amazônia" {
		t.Errorf("Expected Amazônia, got %s", biomeAmazon)
	}

	biomeCerrado := DetectBiome(-15.7939, -47.8828) // Brasília
	if biomeCerrado != "Cerrado" {
		t.Errorf("Expected Cerrado, got %s", biomeCerrado)
	}
}
