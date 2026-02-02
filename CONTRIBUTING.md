# Contributing to Lore

Thank you for your interest in contributing to Lore! This document provides guidelines for contributing to the project.

## Reporting Issues

Before creating an issue, please:

1. Search existing issues to avoid duplicates
2. Use a clear, descriptive title
3. Include steps to reproduce the problem
4. Describe expected vs actual behavior
5. Include relevant logs or screenshots

## Submitting Pull Requests

### Before You Start

1. Open an issue to discuss significant changes before implementing
2. Fork the repository and create a branch from `master`
3. Keep PRs focused on a single change

### Development Setup

See [README.md](README.md) for development environment setup. Quick start:

```bash
# Docker development (recommended)
make dev

# After code changes
make dev-restart
```

### Code Style

This project follows standard Go conventions:

```bash
# Format code
go fmt ./...

# Run linter
go vet ./...
```

Additional guidelines:
- Keep functions focused and small
- Use meaningful variable names
- Add comments only where logic isn't self-evident
- Avoid over-engineering — solve the problem at hand

### Commit Messages

Use [Conventional Commits](https://www.conventionalcommits.org/) format:

```
<type>: <description>

[optional body]
```

Types:
- `feat:` — New feature
- `fix:` — Bug fix
- `refactor:` — Code change that neither fixes a bug nor adds a feature
- `docs:` — Documentation only
- `chore:` — Maintenance tasks
- `test:` — Adding or updating tests

Examples:
```
feat: add reputation graph visualization
fix: correct delegation chain calculation
docs: update README with Docker instructions
refactor: extract common template functions
```

### Testing

Run tests before submitting:

```bash
go test ./...
```

For changes to handlers or services:
- Add or update unit tests
- Use table-driven tests for edge cases
- Follow existing test patterns in the codebase

### Pull Request Process

1. Ensure all tests pass
2. Update documentation if needed
3. Provide a clear description of your changes
4. Request review from maintainers
5. Address review feedback

## Code of Ethics

This project maintains a [Code of Ethics](CODE_OF_ETHICS.md) based on the Rule of St. Benedict. This is a personal commitment by contributors and is not imposed on users of the software.

## Questions?

Open an issue or reach out to the maintainers.
