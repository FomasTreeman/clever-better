-- Rollback: Drop model_metadata table

DROP INDEX IF EXISTS idx_model_metadata_created_at;
DROP INDEX IF EXISTS idx_model_metadata_stage;
DROP INDEX IF EXISTS idx_model_metadata_name_version;
DROP TABLE IF EXISTS model_metadata;
