# Architectural Decision Log

This document records significant architectural decisions made during the development of Clever Better.

## ADR Format

Each decision follows this format:
- **Status**: Proposed | Accepted | Deprecated | Superseded
- **Context**: Why this decision was needed
- **Decision**: What was decided
- **Consequences**: Trade-offs and implications

---

## ADR-001: Monorepo Structure

**Date**: 2024-01-15
**Status**: Accepted

### Context

We need to organize code for three technology stacks:
1. Go backend services
2. Python ML service
3. Terraform infrastructure

Options considered:
- **Polyrepo**: Separate repositories for each component
- **Monorepo**: Single repository with all code

### Decision

Use a **monorepo** structure with Go as the root module and subdirectories for Python and Terraform.

### Consequences

**Positive:**
- Single source of truth for all code
- Atomic commits across components
- Easier dependency management
- Simplified CI/CD pipeline
- Better code discoverability

**Negative:**
- Repository size grows over time
- Different teams need all code locally
- CI must handle multiple languages

**Mitigations:**
- Use sparse checkout for large repos
- Language-specific CI jobs
- Clear directory structure

---

## ADR-002: TimescaleDB for Data Storage

**Date**: 2024-01-15
**Status**: Accepted

### Context

The system needs to store:
- Historical odds data (time-series, high volume)
- Race results and metadata
- Trading history
- ML predictions

Options considered:
- **PostgreSQL**: Standard relational database
- **TimescaleDB**: PostgreSQL with time-series extensions
- **InfluxDB**: Purpose-built time-series database
- **ClickHouse**: Column-oriented analytics database

### Decision

Use **TimescaleDB** as the primary database.

### Consequences

**Positive:**
- PostgreSQL compatibility (familiar SQL, rich ecosystem)
- Automatic partitioning by time (hypertables)
- Excellent compression for time-series data
- Continuous aggregates for pre-computed analytics
- Single database for all data types

**Negative:**
- Requires TimescaleDB extension (not vanilla PostgreSQL)
- Some learning curve for time-series features
- Managed options limited (vs. standard RDS PostgreSQL)

**Mitigations:**
- Use TimescaleDB Cloud or self-host on EC2
- Document time-series specific patterns
- Standard PostgreSQL fallback if needed

---

## ADR-003: ECS Fargate for Container Hosting

**Date**: 2024-01-15
**Status**: Accepted

### Context

Need to deploy containerized services with:
- Automatic scaling
- High availability
- Cost efficiency
- Minimal operational overhead

Options considered:
- **EC2**: Self-managed instances
- **ECS EC2**: Container orchestration on EC2
- **ECS Fargate**: Serverless containers
- **EKS**: Managed Kubernetes

### Decision

Use **ECS Fargate** for all containerized services.

### Consequences

**Positive:**
- No infrastructure management
- Pay only for running containers
- Built-in integration with AWS services
- Simpler than Kubernetes for our scale
- Fast scaling

**Negative:**
- Less control than EC2/EKS
- Slight cold start latency
- Limited to AWS
- More expensive per CPU/memory than EC2

**Mitigations:**
- Use minimum task count to avoid cold starts
- Monitor costs, switch to EC2 if beneficial
- Keep architecture portable (Docker)

---

## ADR-004: Separate Python ML Service

**Date**: 2024-01-15
**Status**: Accepted

### Context

ML functionality could be:
1. Embedded in Go using CGO bindings
2. Separate microservice accessed via API
3. Serverless functions (Lambda)

### Decision

Use a **separate Python microservice** for ML, communicating via gRPC/REST.

### Consequences

**Positive:**
- Best-in-class ML libraries (TensorFlow, PyTorch, scikit-learn)
- Independent scaling of ML workload
- Team can specialize (Go vs Python)
- Easier model updates without redeploying bot
- Better resource isolation

**Negative:**
- Network latency for predictions
- Additional service to maintain
- More complex deployment
- Cross-language debugging harder

**Mitigations:**
- Use gRPC for low-latency communication
- Batch predictions where possible
- Comprehensive logging and tracing
- Shared protobuf definitions

---

## ADR-005: gRPC for Internal Communication

**Date**: 2024-01-15
**Status**: Accepted

### Context

Go services need to communicate with Python ML service for predictions.

Options considered:
- **REST/JSON**: Simple HTTP API
- **gRPC/Protobuf**: Binary RPC protocol
- **Message Queue**: Async via SQS/Redis

### Decision

Use **gRPC with Protocol Buffers** for synchronous Go-Python communication, with REST as fallback.

### Consequences

**Positive:**
- Binary protocol (faster than JSON)
- Strong typing via protobuf schemas
- Streaming support for batch predictions
- Auto-generated client code
- Built-in deadline/timeout handling

**Negative:**
- More complex setup than REST
- Protobuf definitions must be maintained
- Harder to debug than JSON
- Browser incompatibility (internal use only)

**Mitigations:**
- Maintain REST endpoints for debugging
- Use grpcurl for manual testing
- Generate clients from shared proto files

---

## ADR-006: Kelly Criterion for Position Sizing

**Date**: 2024-01-15
**Status**: Accepted

### Context

Need a systematic approach to determine bet sizes that:
- Maximizes long-term growth
- Manages risk appropriately
- Adapts to confidence levels

Options considered:
- **Fixed stake**: Same amount every bet
- **Fixed percentage**: Fixed % of bankroll
- **Kelly Criterion**: Mathematically optimal sizing
- **Fractional Kelly**: Conservative Kelly

### Decision

Use **Fractional Kelly Criterion** (half-Kelly) for position sizing.

### Consequences

**Formula:**
```
Stake = (Bankroll × Edge × Fraction) / (Odds - 1)

Where:
- Edge = (Probability × Odds) - 1
- Fraction = 0.5 (half-Kelly)
```

**Positive:**
- Mathematically optimal for long-term growth
- Automatically sizes based on edge
- Reduces stakes when edge is smaller
- Half-Kelly reduces variance significantly

**Negative:**
- Requires accurate probability estimates
- Can still have significant drawdowns
- May result in very small stakes for marginal edges

**Mitigations:**
- Use half-Kelly to reduce variance
- Cap maximum stake regardless of Kelly
- Minimum stake threshold to avoid tiny bets

---

## ADR-007: Walk-Forward Validation

**Date**: 2024-01-15
**Status**: Accepted

### Context

Need robust validation methodology that:
- Prevents overfitting
- Simulates real trading conditions
- Produces realistic performance estimates

Options considered:
- **Simple train/test split**: One-time split
- **K-fold cross-validation**: Random splits
- **Walk-forward validation**: Rolling temporal splits

### Decision

Use **walk-forward validation** as the primary validation methodology.

### Consequences

**Implementation:**
```
Time: |----Train----|--Val--|--Test--|----Train----|--Val--|--Test--|
Window 1: [Jan-Jun] [Jul] [Aug]
Window 2: [Feb-Jul] [Aug] [Sep]
...
```

**Positive:**
- No future data leakage
- Tests on truly out-of-sample data
- Simulates real trading deployment
- Reveals strategy decay over time

**Negative:**
- Less training data per model
- More complex to implement
- Longer to run than simple split
- Multiple models trained per backtest

**Mitigations:**
- Use sufficient window sizes
- Parallelize window training
- Cache intermediate results

---

## ADR-008: Feature Store Pattern

**Date**: 2024-01-15
**Status**: Proposed

### Context

Features are computed from raw data and used by:
- Model training (offline)
- Real-time prediction (online)

Need consistency between training and serving features.

### Decision

Implement a **simple feature store** using TimescaleDB materialized views for offline features and computed features for online serving.

### Consequences

**Positive:**
- Consistent features between training and serving
- Pre-computed features for faster training
- Audit trail of feature values
- Easier feature discovery and reuse

**Negative:**
- Additional storage for materialized features
- Refresh overhead for continuous aggregates
- Need to handle feature drift

**Mitigations:**
- Use TimescaleDB continuous aggregates
- Monitor feature statistics over time
- Document feature definitions clearly

---

## Decision Template

Use this template for new decisions:

```markdown
## ADR-XXX: [Title]

**Date**: YYYY-MM-DD
**Status**: Proposed | Accepted | Deprecated | Superseded

### Context

[What is the issue? Why does this decision need to be made?]

Options considered:
- **Option A**: Description
- **Option B**: Description

### Decision

[What was decided and why?]

### Consequences

**Positive:**
-

**Negative:**
-

**Mitigations:**
-
```
