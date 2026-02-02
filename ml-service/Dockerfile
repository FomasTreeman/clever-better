# Multi-stage build for ML service with TensorFlow and PyTorch support
FROM python:3.11-slim AS build

RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    libpq-dev \
    libgomp1 \
    git \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY requirements.txt requirements-dev.txt ./
RUN pip install --no-cache-dir -r requirements.txt

COPY proto ./proto
COPY app ./app

RUN python -m grpc_tools.protoc \
    -I proto \
    --python_out=app/generated \
    --grpc_python_out=app/generated \
    proto/ml_service.proto

FROM python:3.11-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    libpq-dev \
    supervisor \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=build /usr/local /usr/local
COPY --from=build /app /app
COPY supervisord.conf /app/supervisord.conf

ENV PYTHONUNBUFFERED=1
ENV PYTHONPATH=/app
ENV TF_CPP_MIN_LOG_LEVEL=2
ENV OMP_NUM_THREADS=4

# Expose FastAPI, gRPC, and MLflow ports
EXPOSE 8000 50051 5000

HEALTHCHECK --interval=30s --timeout=5s --retries=3 CMD python -c "import urllib.request; urllib.request.urlopen('http://localhost:8000/health').read()" || exit 1

CMD ["supervisord", "-c", "/app/supervisord.conf"]
