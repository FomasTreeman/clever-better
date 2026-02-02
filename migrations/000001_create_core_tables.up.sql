-- Create core tables for races and runners
-- This migration establishes the foundational tables for race data

CREATE TABLE IF NOT EXISTS races (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scheduled_start TIMESTAMPTZ NOT NULL,
    actual_start TIMESTAMPTZ,
    track VARCHAR(255) NOT NULL,
    race_type VARCHAR(255) NOT NULL,
    distance INT NOT NULL,
    grade VARCHAR(50),
    conditions JSONB,
    status VARCHAR(50) DEFAULT 'scheduled',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_races_scheduled_start ON races(scheduled_start DESC);
CREATE INDEX idx_races_track ON races(track);
CREATE INDEX idx_races_status ON races(status);
CREATE INDEX idx_races_created_at ON races(created_at DESC);

CREATE TABLE IF NOT EXISTS runners (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    race_id UUID NOT NULL REFERENCES races(id) ON DELETE CASCADE,
    trap_number INT NOT NULL,
    name VARCHAR(255) NOT NULL,
    form_rating DECIMAL(10, 2),
    weight DECIMAL(10, 2),
    trainer VARCHAR(255),
    days_since_last_race INT,
    metadata JSONB,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_runners_race_id ON runners(race_id);
CREATE INDEX idx_runners_trap_number ON runners(trap_number);
CREATE INDEX idx_runners_created_at ON runners(created_at DESC);
CREATE UNIQUE INDEX idx_runners_race_trap ON runners(race_id, trap_number);
