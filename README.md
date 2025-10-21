# Temporal Terraform Provider

## Overview

The Temporal Terraform Provider allows you to manage [Temporal](https://temporal.io/) resources using [Terraform](https://www.terraform.io/).

## Usage

See usage and examples on <https://registry.terraform.io/providers/stan-stately/temporal>.

## Releasing
- Push a git tag with the new version:
```
git checkout main
git tag v1.1.1
git push origin --tags
```

### Dev tools

```bash
# Build the provider
make build

# Run linter
make lint

# Run tests
make test

# Run acceptance tests
make testacc

# Clean up
make clean
```
