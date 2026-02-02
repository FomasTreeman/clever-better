# System Architecture

This document provides a comprehensive overview of the Clever Better system architecture, including component design, interaction patterns, and technology rationale.

## Table of Contents

- [Overview](#overview)
- [System Architecture Diagram](#system-architecture-diagram)
- [Core Components](#core-components)
- [Component Interactions](#component-interactions)
- [Technology Stack Rationale](#technology-stack-rationale)
- [Data Architecture](#data-architecture)
- [Scalability Considerations](#scalability-considerations)

## Overview

Clever Better is designed as a modular, microservices-inspired architecture with clear separation of concerns:

- **Go Backend**: Handles all trading operations, backtesting, and data ingestion
- **Python ML Service**: Dedicated to machine learning model training and inference
- **TimescaleDB**: Optimized time-series database for historical and real-time data
- **External Integrations**: Betfair API and greyhound racing data sources

## System Architecture Diagram

```mermaid
graph TB
    subgraph "Client Layer"
        CLI[CLI Tools]
        API[REST/gRPC API]
    end

    subgraph "Go Backend Services"
        Bot[Trading Bot<br/>cmd/bot]
        Backtest[Backtesting Engine<br/>cmd/backtest]
        DataIngest[Data Ingestion<br/>cmd/data-ingestion]

        subgraph "Internal Packages"
            BetfairClient[Betfair Client<br/>internal/betfair]
            Strategy[Strategy Engine<br/>internal/strategy]
            Repository[Data Repository<br/>internal/repository]
            MLClient[ML Client<br/>internal/ml]
        end
    end

    subgraph "Python ML Service"
        MLService[ML API Server<br/>FastAPI]
        FeatureEng[Feature Engineering]
        ModelTraining[Model Training]
        Inference[Prediction Service]
    end

    subgraph "Data Layer"
        TimescaleDB[(TimescaleDB<br/>Time-Series Data)]
        ModelStorage[(Model Storage<br/>S3/Local)]
    end

    subgraph "External Services"
        BetfairAPI[Betfair Exchange API]
        BetfairStream[Betfair Streaming API]
        DataSources[Racing Data Sources]
    end

    CLI --> Bot
    CLI --> Backtest
    API --> Bot

    Bot --> BetfairClient
    Bot --> Strategy
    Bot --> MLClient
    Bot --> Repository

    Backtest --> Strategy
    Backtest --> MLClient
    Backtest --> Repository

    DataIngest --> Repository
    DataIngest --> DataSources

    BetfairClient --> BetfairAPI
    BetfairClient --> BetfairStream

    MLClient --> MLService
    MLService --> FeatureEng
    MLService --> ModelTraining
    MLService --> Inference
    ModelTraining --> ModelStorage

    Repository --> TimescaleDB
    FeatureEng --> TimescaleDB
    Inference --> TimescaleDB
```

## Core Components

### Trading Bot (`cmd/bot`)

The main application responsible for live trading operations.

**Responsibilities:**
- Connect to Betfair Exchange API
- Subscribe to market streaming data
- Request predictions from ML service
- Execute betting strategies with risk management
- Log all trading activity

**Key Internal Dependencies:**
- `internal/betfair` - Betfair API client
- `internal/strategy` - Strategy execution engine
- `internal/ml` - ML service client
- `internal/config` - Configuration management
- `internal/metrics` - Performance metrics

### Backtesting Engine (`cmd/backtest`)

CLI tool for historical strategy validation.

**Responsibilities:**
- Load historical race and odds data
- Simulate trading strategies against historical data
- Run Monte Carlo simulations for risk analysis
- Perform walk-forward optimization
- Generate performance reports

**Key Internal Dependencies:**
- `internal/backtest` - Core backtesting logic
- `internal/strategy` - Strategy interfaces
- `internal/repository` - Historical data access

### Data Ingestion Service (`cmd/data-ingestion`)

Service for collecting and storing racing data.

**Responsibilities:**
- Fetch historical race results
- Collect greyhound form data
- Store odds snapshots
- Maintain data quality and consistency

**Key Internal Dependencies:**
- `internal/datasource` - External data source clients
- `internal/repository` - Data persistence

### ML Service (`ml-service/`)

Python microservice for machine learning operations.

**Responsibilities:**
- Feature engineering from raw data
- Model training and validation
- Real-time prediction serving
- Model versioning and management

**Technology Stack:**
- FastAPI for HTTP/gRPC API
- TensorFlow/PyTorch for deep learning
- scikit-learn for classical ML
- pandas/numpy for data processing

## Component Interactions

### Live Trading Flow

```mermaid
sequenceDiagram
    participant Bot as Trading Bot
    participant Betfair as Betfair API
    participant ML as ML Service
    participant DB as TimescaleDB

    Bot->>Betfair: Subscribe to market stream
    Betfair-->>Bot: Market update

    Bot->>DB: Fetch race context
    DB-->>Bot: Race data

    Bot->>ML: Request prediction
    ML->>DB: Fetch features
    DB-->>ML: Feature data
    ML-->>Bot: Prediction + confidence

    alt Confidence > threshold
        Bot->>Bot: Calculate stake (Kelly criterion)
        Bot->>Betfair: Place bet
        Betfair-->>Bot: Bet confirmation
        Bot->>DB: Log trade
    end
```

### Backtesting Flow

```mermaid
sequenceDiagram
    participant CLI as Backtest CLI
    participant Engine as Backtest Engine
    participant ML as ML Service
    participant DB as TimescaleDB

    CLI->>Engine: Start backtest
    Engine->>DB: Load historical data
    DB-->>Engine: Race/odds data

    loop For each historical race
        Engine->>ML: Get prediction (historical features)
        ML-->>Engine: Prediction
        Engine->>Engine: Simulate trade
        Engine->>Engine: Update portfolio
    end

    Engine->>Engine: Calculate metrics
    Engine-->>CLI: Performance report
```

## Technology Stack Rationale

### Why Go for Backend?

- **Performance**: Low latency critical for live trading
- **Concurrency**: Goroutines ideal for handling multiple market streams
- **Type Safety**: Compile-time checks reduce runtime errors
- **Single Binary**: Easy deployment without runtime dependencies
- **Strong Standard Library**: Excellent HTTP, JSON, and networking support

### Why Python for ML?

- **ML Ecosystem**: TensorFlow, PyTorch, scikit-learn mature and well-supported
- **Data Science Tools**: pandas, numpy, matplotlib industry standard
- **Rapid Prototyping**: Quick iteration on model experiments
- **Research Compatibility**: Easy to incorporate academic research

### Why TimescaleDB?

- **Time-Series Optimization**: Purpose-built for time-series data
- **PostgreSQL Compatibility**: Familiar SQL interface, rich ecosystem
- **Compression**: Efficient storage for large historical datasets
- **Continuous Aggregates**: Pre-computed rollups for fast queries
- **Retention Policies**: Automatic data lifecycle management

### Why gRPC for Go-Python Communication?

- **Performance**: Binary protocol faster than JSON/REST
- **Type Safety**: Protobuf schemas ensure contract compliance
- **Streaming**: Native support for bidirectional streaming
- **Code Generation**: Auto-generated clients reduce boilerplate

## Data Architecture

### Core Data Entities

```mermaid
erDiagram
    Race ||--o{ Runner : has
    Race ||--o{ OddsSnapshot : has
    Runner ||--o{ OddsSnapshot : for
    Race ||--o{ Trade : on
    Runner ||--o{ Trade : for
    Strategy ||--o{ Trade : uses
    Model ||--o{ Prediction : makes

    Race {
        uuid id PK
        timestamp scheduled_start
        string track
        string race_type
        int distance
        string conditions
    }

    Runner {
        uuid id PK
        uuid race_id FK
        int trap_number
        string name
        float form_rating
        jsonb metadata
    }

    OddsSnapshot {
        uuid id PK
        uuid race_id FK
        uuid runner_id FK
        timestamp captured_at
        float back_price
        float lay_price
        float back_size
        float lay_size
    }

    Trade {
        uuid id PK
        uuid race_id FK
        uuid runner_id FK
        uuid strategy_id FK
        string side
        float odds
        float stake
        string status
        float profit_loss
        timestamp executed_at
    }

    Strategy {
        uuid id PK
        string name
        jsonb parameters
        boolean active
    }

    Model {
        uuid id PK
        string name
        string version
        string path
        jsonb metrics
        timestamp trained_at
    }

    Prediction {
        uuid id PK
        uuid model_id FK
        uuid race_id FK
        uuid runner_id FK
        float probability
        float confidence
        timestamp predicted_at
    }
```

### Data Partitioning Strategy

TimescaleDB hypertables partition data by time:

- **OddsSnapshot**: Partitioned by week, retained for 2 years
- **Trade**: Partitioned by month, retained indefinitely
- **Prediction**: Partitioned by day, retained for 6 months

## Scalability Considerations

### Horizontal Scaling

- **Trading Bot**: Single instance per Betfair account (API limits)
- **ML Service**: Can scale horizontally behind load balancer
- **Data Ingestion**: Can parallelize by data source

### Performance Optimizations

- **Connection Pooling**: Database connection pools sized for workload
- **Caching**: Redis layer for frequently accessed reference data
- **Async Processing**: Non-critical operations processed asynchronously
- **Batch Queries**: Bulk inserts for high-volume data ingestion

### Monitoring Points

- API latency percentiles (p50, p95, p99)
- ML inference time
- Database query performance
- Market data lag
- Trade execution success rate
