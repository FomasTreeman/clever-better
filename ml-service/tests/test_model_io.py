"""Tests for model persistence and I/O operations."""
import pytest
import os
import tempfile
import pickle
import json
from pathlib import Path


class TestModelSaving:
    """Test model saving operations."""

    def test_save_model_pickle(self):
        """Test saving model in pickle format."""
        from app.models import save_model

        # Mock model
        class MockModel:
            def __init__(self):
                self.weights = [0.1, 0.2, 0.3]

        model = MockModel()

        with tempfile.TemporaryDirectory() as tmpdir:
            model_path = Path(tmpdir) / "model.pkl"

            save_model(model, str(model_path), format="pickle")

            assert model_path.exists()
            assert model_path.stat().st_size > 0

    def test_save_model_with_metadata(self):
        """Test saving model with metadata."""
        from app.models import save_model_with_metadata

        class MockModel:
            def __init__(self):
                self.weights = [0.1, 0.2, 0.3]

        model = MockModel()
        metadata = {
            "model_type": "race_predictor",
            "version": "1.0.0",
            "created_at": "2026-02-15T10:00:00Z",
            "accuracy": 0.85,
        }

        with tempfile.TemporaryDirectory() as tmpdir:
            save_model_with_metadata(model, str(tmpdir), metadata)

            # Check model file exists
            assert (Path(tmpdir) / "model.pkl").exists()

            # Check metadata file exists
            metadata_path = Path(tmpdir) / "metadata.json"
            assert metadata_path.exists()

            # Validate metadata
            with open(metadata_path) as f:
                saved_metadata = json.load(f)

            assert saved_metadata["model_type"] == "race_predictor"
            assert saved_metadata["version"] == "1.0.0"

    def test_save_model_version_tracking(self):
        """Test model versioning."""
        from app.models import save_versioned_model

        class MockModel:
            def __init__(self):
                self.weights = [0.1, 0.2, 0.3]

        model = MockModel()

        with tempfile.TemporaryDirectory() as tmpdir:
            # Save multiple versions
            v1_path = save_versioned_model(model, str(tmpdir), version="v1")
            v2_path = save_versioned_model(model, str(tmpdir), version="v2")

            assert Path(v1_path).exists()
            assert Path(v2_path).exists()
            assert "v1" in v1_path
            assert "v2" in v2_path


class TestModelLoading:
    """Test model loading operations."""

    def test_load_model_pickle(self):
        """Test loading model from pickle file."""
        from app.models import save_model, load_model

        class MockModel:
            def __init__(self):
                self.weights = [0.1, 0.2, 0.3]

        original_model = MockModel()

        with tempfile.TemporaryDirectory() as tmpdir:
            model_path = Path(tmpdir) / "model.pkl"

            save_model(original_model, str(model_path), format="pickle")
            loaded_model = load_model(str(model_path), format="pickle")

            assert loaded_model.weights == original_model.weights

    def test_load_model_with_metadata(self):
        """Test loading model with metadata."""
        from app.models import save_model_with_metadata, load_model_with_metadata

        class MockModel:
            def __init__(self):
                self.weights = [0.1, 0.2, 0.3]

        original_model = MockModel()
        original_metadata = {
            "model_type": "race_predictor",
            "accuracy": 0.85,
        }

        with tempfile.TemporaryDirectory() as tmpdir:
            save_model_with_metadata(original_model, str(tmpdir), original_metadata)

            loaded_model, loaded_metadata = load_model_with_metadata(str(tmpdir))

            assert loaded_model.weights == original_model.weights
            assert loaded_metadata["model_type"] == "race_predictor"
            assert loaded_metadata["accuracy"] == 0.85

    def test_load_nonexistent_model(self):
        """Test loading nonexistent model raises error."""
        from app.models import load_model

        with pytest.raises(FileNotFoundError):
            load_model("/nonexistent/path/model.pkl")

    def test_load_corrupted_model(self):
        """Test loading corrupted model file."""
        from app.models import load_model

        with tempfile.TemporaryDirectory() as tmpdir:
            model_path = Path(tmpdir) / "corrupted.pkl"

            # Write corrupted data
            with open(model_path, "wb") as f:
                f.write(b"not a valid pickle file")

            with pytest.raises(Exception):  # Could be pickle.UnpicklingError or similar
                load_model(str(model_path))


class TestModelRegistry:
    """Test model registry operations."""

    def test_register_model(self):
        """Test registering model in registry."""
        from app.models import ModelRegistry

        registry = ModelRegistry()

        class MockModel:
            def __init__(self):
                self.weights = [0.1, 0.2, 0.3]

        model = MockModel()
        model_id = registry.register(model, metadata={
            "model_type": "race_predictor",
            "version": "1.0.0"
        })

        assert model_id is not None
        assert registry.exists(model_id)

    def test_retrieve_model_from_registry(self):
        """Test retrieving model from registry."""
        from app.models import ModelRegistry

        registry = ModelRegistry()

        class MockModel:
            def __init__(self):
                self.weights = [0.1, 0.2, 0.3]

        model = MockModel()
        model_id = registry.register(model)

        retrieved_model = registry.get(model_id)

        assert retrieved_model is not None
        assert retrieved_model.weights == model.weights

    def test_list_registered_models(self):
        """Test listing all registered models."""
        from app.models import ModelRegistry

        registry = ModelRegistry()

        class MockModel:
            def __init__(self):
                self.weights = [0.1, 0.2, 0.3]

        # Register multiple models
        model1 = MockModel()
        model2 = MockModel()

        registry.register(model1, metadata={"version": "1.0.0"})
        registry.register(model2, metadata={"version": "2.0.0"})

        models = registry.list_models()

        assert len(models) >= 2

    def test_delete_model_from_registry(self):
        """Test deleting model from registry."""
        from app.models import ModelRegistry

        registry = ModelRegistry()

        class MockModel:
            def __init__(self):
                self.weights = [0.1, 0.2, 0.3]

        model = MockModel()
        model_id = registry.register(model)

        assert registry.exists(model_id)

        registry.delete(model_id)

        assert not registry.exists(model_id)


class TestModelCheckpointing:
    """Test model checkpointing during training."""

    def test_save_checkpoint(self):
        """Test saving training checkpoint."""
        from app.models import save_checkpoint

        checkpoint_data = {
            "epoch": 5,
            "model_state": {"weights": [0.1, 0.2]},
            "optimizer_state": {"lr": 0.01},
            "loss": 0.35,
        }

        with tempfile.TemporaryDirectory() as tmpdir:
            checkpoint_path = Path(tmpdir) / "checkpoint.pt"

            save_checkpoint(checkpoint_data, str(checkpoint_path))

            assert checkpoint_path.exists()

    def test_load_checkpoint(self):
        """Test loading training checkpoint."""
        from app.models import save_checkpoint, load_checkpoint

        original_checkpoint = {
            "epoch": 5,
            "model_state": {"weights": [0.1, 0.2]},
            "loss": 0.35,
        }

        with tempfile.TemporaryDirectory() as tmpdir:
            checkpoint_path = Path(tmpdir) / "checkpoint.pt"

            save_checkpoint(original_checkpoint, str(checkpoint_path))
            loaded_checkpoint = load_checkpoint(str(checkpoint_path))

            assert loaded_checkpoint["epoch"] == 5
            assert loaded_checkpoint["loss"] == 0.35

    def test_checkpoint_overwrite_prevention(self):
        """Test preventing accidental checkpoint overwrite."""
        from app.models import save_checkpoint

        checkpoint_data = {
            "epoch": 5,
            "loss": 0.35,
        }

        with tempfile.TemporaryDirectory() as tmpdir:
            checkpoint_path = Path(tmpdir) / "checkpoint.pt"

            # Save first checkpoint
            save_checkpoint(checkpoint_data, str(checkpoint_path))

            # Attempt to overwrite without force flag
            with pytest.raises(FileExistsError):
                save_checkpoint(checkpoint_data, str(checkpoint_path), overwrite=False)


class TestModelExport:
    """Test exporting models to different formats."""

    def test_export_to_onnx(self):
        """Test exporting model to ONNX format."""
        from app.models import export_to_onnx

        class MockModel:
            def __init__(self):
                self.weights = [0.1, 0.2, 0.3]

        model = MockModel()

        with tempfile.TemporaryDirectory() as tmpdir:
            onnx_path = Path(tmpdir) / "model.onnx"

            # This would require torch or tf installed
            # export_to_onnx(model, str(onnx_path))
            # assert onnx_path.exists()
            pass  # Skip actual export in test

    def test_export_model_metadata(self):
        """Test exporting model metadata separately."""
        from app.models import export_metadata

        metadata = {
            "model_type": "race_predictor",
            "version": "1.0.0",
            "features": ["odds", "distance", "going"],
            "accuracy": 0.85,
        }

        with tempfile.TemporaryDirectory() as tmpdir:
            metadata_path = Path(tmpdir) / "metadata.json"

            export_metadata(metadata, str(metadata_path))

            assert metadata_path.exists()

            with open(metadata_path) as f:
                exported = json.load(f)

            assert exported["model_type"] == "race_predictor"
            assert len(exported["features"]) == 3


class TestModelCompression:
    """Test model compression for deployment."""

    def test_compress_model(self):
        """Test compressing model file."""
        from app.models import compress_model

        class MockModel:
            def __init__(self):
                self.weights = [0.1, 0.2, 0.3] * 1000  # Larger model

        model = MockModel()

        with tempfile.TemporaryDirectory() as tmpdir:
            model_path = Path(tmpdir) / "model.pkl"
            compressed_path = Path(tmpdir) / "model.pkl.gz"

            # Save then compress
            with open(model_path, "wb") as f:
                pickle.dump(model, f)

            compress_model(str(model_path), str(compressed_path))

            assert compressed_path.exists()
            # Compressed should be smaller
            assert compressed_path.stat().st_size < model_path.stat().st_size

    def test_decompress_model(self):
        """Test decompressing model file."""
        from app.models import compress_model, decompress_model

        class MockModel:
            def __init__(self):
                self.weights = [0.1, 0.2, 0.3]

        model = MockModel()

        with tempfile.TemporaryDirectory() as tmpdir:
            model_path = Path(tmpdir) / "model.pkl"
            compressed_path = Path(tmpdir) / "model.pkl.gz"
            decompressed_path = Path(tmpdir) / "model_decompressed.pkl"

            # Save, compress, decompress
            with open(model_path, "wb") as f:
                pickle.dump(model, f)

            compress_model(str(model_path), str(compressed_path))
            decompress_model(str(compressed_path), str(decompressed_path))

            assert decompressed_path.exists()

            # Verify model integrity
            with open(decompressed_path, "rb") as f:
                loaded_model = pickle.load(f)

            assert loaded_model.weights == model.weights
