-- Enable PostGIS extension
CREATE EXTENSION IF NOT EXISTS postgis;

-- Table: species
CREATE TABLE IF NOT EXISTS species (
    id BIGINT PRIMARY KEY,
    taxon_key BIGINT UNIQUE NOT NULL,
    species_name VARCHAR(255) NOT NULL,
    scientific_name VARCHAR(255) NOT NULL,
    kingdom VARCHAR(100),
    phylum VARCHAR(100),
    class VARCHAR(100),
    order_name VARCHAR(100),
    family VARCHAR(100),
    iucn_status VARCHAR(10) NOT NULL DEFAULT 'LC',
    description TEXT,
    image_url TEXT,
    total_count BIGINT DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Table: observations
CREATE TABLE IF NOT EXISTS observations (
    id BIGINT PRIMARY KEY,
    taxon_key BIGINT NOT NULL,
    species_name VARCHAR(255) NOT NULL,
    scientific_name VARCHAR(255) NOT NULL,
    latitude DOUBLE PRECISION NOT NULL,
    longitude DOUBLE PRECISION NOT NULL,
    location_geom GEOMETRY(Point, 4326),
    image_url TEXT,
    event_date TIMESTAMP WITH TIME ZONE NOT NULL,
    biome VARCHAR(100) NOT NULL,
    country VARCHAR(100),
    locality TEXT,
    dataset_key VARCHAR(255),
    iucn_status VARCHAR(10) DEFAULT 'LC',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Spatial and standard indexes
CREATE INDEX IF NOT EXISTS idx_obs_location_geom ON observations USING GIST (location_geom);
CREATE INDEX IF NOT EXISTS idx_obs_biome ON observations(biome);
CREATE INDEX IF NOT EXISTS idx_obs_taxon_key ON observations(taxon_key);
CREATE INDEX IF NOT EXISTS idx_obs_iucn_status ON observations(iucn_status);

-- Trigger to keep location_geom in sync
CREATE OR REPLACE FUNCTION update_observation_geom()
RETURNS TRIGGER AS $$
BEGIN
    NEW.location_geom = ST_SetSRID(ST_MakePoint(NEW.longitude, NEW.latitude), 4326);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_update_obs_geom ON observations;
CREATE TRIGGER trg_update_obs_geom
BEFORE INSERT OR UPDATE ON observations
FOR EACH ROW EXECUTE FUNCTION update_observation_geom();
