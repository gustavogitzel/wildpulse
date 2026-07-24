-- Migration 000002: Ensure auto-increment sequences exist for species and observations tables
CREATE SEQUENCE IF NOT EXISTS species_id_seq;
ALTER TABLE species ALTER COLUMN id SET DEFAULT nextval('species_id_seq');

CREATE SEQUENCE IF NOT EXISTS observations_id_seq;
ALTER TABLE observations ALTER COLUMN id SET DEFAULT nextval('observations_id_seq');
