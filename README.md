# 3-Tier Demo Service Provider

DCM service provider for demonstrating 3-tier (web, app, db) stack lifecycle.

## Overview

This service provider implements the DCM interface for 3-tier demo stacks, with an AEP-compliant REST API.

## Development

### Prerequisites

- Go 1.25 or later
- Make
- [Spectral](https://stoplight.io/open-source/spectral) (for `make check-aep`)

### Build and run

```bash
make build
make run
```

### Code generation

Code is generated from `api/v1alpha1/openapi.yaml`:

```bash
make generate-api
make check-generate-api   # verify generated files are in sync
```

### Test and lint

```bash
make test
make fmt vet
make check-aep
```

## License

Apache License 2.0 - see [LICENSE](LICENSE).
