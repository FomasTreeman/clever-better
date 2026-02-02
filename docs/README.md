# Clever Better Documentation

Welcome to the Clever Better documentation. This directory contains comprehensive documentation for the greyhound racing betting bot system.

## Table of Contents

### Core Documentation

| Document | Description |
|----------|-------------|
| [ARCHITECTURE.md](ARCHITECTURE.md) | System architecture, component design, and technology rationale |
| [INFRASTRUCTURE.md](INFRASTRUCTURE.md) | AWS infrastructure design, network topology, and security |
| [DATA_FLOW.md](DATA_FLOW.md) | Data processing pipelines and flow diagrams |
| [ML_STRATEGY.md](ML_STRATEGY.md) | Machine learning approach and strategy discovery methodology |
| [BACKTESTING.md](BACKTESTING.md) | Backtesting methodology, metrics, and validation approach |

### Operational Documentation

| Document | Description |
|----------|-------------|
| [API_REFERENCE.md](API_REFERENCE.md) | API endpoints and integration documentation |
| [DEPLOYMENT.md](DEPLOYMENT.md) | Deployment procedures and runbooks |
| [SECURITY.md](SECURITY.md) | Security considerations and best practices |
| [DEVELOPMENT.md](DEVELOPMENT.md) | Development environment setup and guidelines |
| [TROUBLESHOOTING.md](TROUBLESHOOTING.md) | Common issues and their solutions |

### AI Context Documentation

The [ai-context/](ai-context/) directory contains documentation optimized for AI-assisted development:

| Document | Description |
|----------|-------------|
| [PROJECT_OVERVIEW.md](ai-context/PROJECT_OVERVIEW.md) | High-level project summary for AI consumption |
| [COMPONENT_GUIDE.md](ai-context/COMPONENT_GUIDE.md) | Detailed component responsibilities |
| [CONVENTIONS.md](ai-context/CONVENTIONS.md) | Coding standards and patterns |
| [DECISION_LOG.md](ai-context/DECISION_LOG.md) | Architectural Decision Records (ADRs) |

### Diagrams

The [diagrams/](diagrams/) directory contains Mermaid source files for all architectural diagrams:

- `system-architecture.mmd` - Overall system architecture
- `data-flow.mmd` - Data flow and processing pipeline
- `ml-pipeline.mmd` - ML training and inference pipeline
- `deployment-architecture.mmd` - AWS deployment topology
- `backtesting-workflow.mmd` - Backtesting process flow

## Documentation Guidelines

### Updating Documentation

1. **Keep documentation in sync with code** - Update docs when making significant changes
2. **Use Mermaid for diagrams** - Store diagram source in `diagrams/` directory
3. **Follow the template structure** - Maintain consistency across documents
4. **Link between documents** - Use relative links for cross-references

### Generating Diagrams

To generate PNG images from Mermaid files:

```bash
make docs-diagrams
```

### Serving Documentation Locally

To serve documentation with a local web server:

```bash
make docs-serve
```

Then open http://localhost:8000 in your browser.

## Quick Links

- [Getting Started](../README.md#quick-start)
- [Contributing Guidelines](../CONTRIBUTING.md)
- [Project License](../LICENSE)
