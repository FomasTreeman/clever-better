# Contributing to Clever Better

Thank you for your interest in contributing to Clever Better! This document provides guidelines and information for contributors.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [Coding Standards](#coding-standards)
- [Testing Requirements](#testing-requirements)
- [Documentation Standards](#documentation-standards)
- [Pull Request Process](#pull-request-process)

## Code of Conduct

- Be respectful and inclusive
- Focus on constructive feedback
- Help maintain a positive environment

## Getting Started

1. Fork the repository
2. Clone your fork locally
3. Set up the development environment (see README.md)
4. Create a feature branch from `main`

## Development Workflow

1. Create a feature branch: `git checkout -b feature/your-feature-name`
2. Make your changes following coding standards
3. Write/update tests as needed
4. Run the test suite: `make test`
5. Run linters: `make lint`
6. Commit with clear messages
7. Push to your fork
8. Open a Pull Request

## Coding Standards

### Go Code

- Follow the [Effective Go](https://golang.org/doc/effective_go) guidelines
- Use `gofmt` and `goimports` for formatting
- Run `golangci-lint` before committing
- Write meaningful error messages
- Add comments for exported functions and types

### Python Code

- Follow PEP 8 style guidelines
- Use type hints for function signatures
- Use `black` for formatting
- Use `isort` for import sorting
- Run `flake8` and `mypy` before committing

### General

- Keep functions focused and small
- Write self-documenting code
- Avoid premature optimization
- Handle errors explicitly

## Testing Requirements

- All new features must include tests
- Bug fixes should include regression tests
- Maintain or improve code coverage
- Run the full test suite before submitting PRs

### Running Tests

```bash
# Run all tests
make test

# Run Go tests only
make go-test

# Run Python tests only
make py-test

# Run integration tests
make test-integration
```

## Documentation Standards

- Update relevant documentation for new features
- Include docstrings for public APIs
- Add architecture decision records for significant changes
- Keep README.md up to date

## Pull Request Process

1. Ensure all tests pass
2. Update documentation as needed
3. Fill out the PR template completely
4. Request review from maintainers
5. Address review feedback promptly
6. Squash commits before merging (if requested)

### Commit Message Format

Use clear, descriptive commit messages:

```
<type>(<scope>): <short description>

<longer description if needed>

<footer with issue references>
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`

Example:
```
feat(backtest): add Monte Carlo simulation support

Implements Monte Carlo simulation for strategy backtesting with
configurable number of iterations and confidence intervals.

Closes #123
```

## Questions?

If you have questions, please open an issue for discussion before starting work on large changes.
