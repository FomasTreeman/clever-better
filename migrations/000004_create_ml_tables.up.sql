-- Create ML-related tables for models and predictions

CREATE TABLE IF NOT EXISTS models (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    version VARCHAR(50) NOT NULL,
    model_type VARCHAR(100) NOT NULL,
    path VARCHAR(500) NOT NULL,
    metrics JSONB,
    hyperparameters JSONB,
    trained_at TIMESTAMPTZ NOT NULL,
    active BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(name, version)
);

CREATE INDEX idx_models_active ON models(active);
CREATE INDEX idx_models_trained_at ON models(trained_at DESC);
CREATE INDEX idx_models_created_at ON models(created_at DESC);

CREATE TABLE IF NOT EXISTS predictions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    model_id UUID NOT NULL REFERENCES models(id) ON DELETE RESTRICT,
    race_id UUID NOT NULL REFERENCES races(id) ON DELETE CASCADE,
    runner_id UUID NOT NULL REFERENCES runners(id) ON DELETE CASCADE,
    probability DECIMAL(5, 4) NOT NULL,
    confidence DECIMAL(5, 4) NOT NULL,
    features JSONB,
    predicted_at TIMESTAMPTZ NOT NULL
);

SELECT create_hypertable('predictions', 'predicted_at', if_not_exists => TRUE);

CREATE INDEX idx_predictions_model_id ON predictions(model_id, predicted_at DESC);
CREATE INDEX idx_predictions_race_id ON predictions(race_id, predicted_at DESC);
CREATE INDEX idx_predictions_runner_id ON predictions(runner_id, predicted_at DESC);

-- Add compression policy (compress after 7 days)
SELECT add_compression_policy('predictions', INTERVAL '7 days', if_not_exists => TRUE);

-- Add retention policy (keep for 1 year)
SELECT add_retention_policy('predictions', INTERVAL '1 year', if_not_exists => TRUE);
