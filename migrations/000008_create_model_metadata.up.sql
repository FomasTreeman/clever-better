-- Migration: Add model_metadata table for tracking ML models

CREATE TABLE IF NOT EXISTS model_metadata (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    model_name VARCHAR(255) NOT NULL,
    model_type VARCHAR(64) NOT NULL,
    version VARCHAR(32) NOT NULL,
    mlflow_run_id VARCHAR(255) NOT NULL,
    stage VARCHAR(32) NOT NULL DEFAULT 'None',
    metrics JSONB,
    hyperparameters JSONB,
    feature_names JSONB,
    training_dataset_size INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_model_metadata_name_version ON model_metadata(model_name, version);
CREATE INDEX idx_model_metadata_stage ON model_metadata(stage);
CREATE INDEX idx_model_metadata_created_at ON model_metadata(created_at DESC);

COMMENT ON TABLE model_metadata IS 'Tracks ML model training runs and versions';
COMMENT ON COLUMN model_metadata.model_name IS 'Name of the model (e.g., ensemble, classifier, rl_agent)';
COMMENT ON COLUMN model_metadata.model_type IS 'Type of model (sklearn, tensorflow, pytorch)';
COMMENT ON COLUMN model_metadata.version IS 'Model version number';
COMMENT ON COLUMN model_metadata.mlflow_run_id IS 'MLflow run ID for tracking';
COMMENT ON COLUMN model_metadata.stage IS 'Deployment stage (None, Staging, Production, Archived)';
