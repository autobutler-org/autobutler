# Quark

Your own private cloud, running in your house. Photos, files, documents — all on hardware you own, off servers you don't trust.

[![CI - Android](https://github.com/autobutler-org/quark/actions/workflows/ci-android.yml/badge.svg)](https://github.com/autobutler-org/quark/actions/workflows/ci-android.yml)
[![CI - Backend](https://github.com/autobutler-org/quark/actions/workflows/ci-backend.yml/badge.svg)](https://github.com/autobutler-org/quark/actions/workflows/ci-backend.yml)
[![CI - iOS](https://github.com/autobutler-org/quark/actions/workflows/ci-ios.yml/badge.svg)](https://github.com/autobutler-org/quark/actions/workflows/ci-ios.yml)
[![CI - Web](https://github.com/autobutler-org/quark/actions/workflows/ci-web.yml/badge.svg)](https://github.com/autobutler-org/quark/actions/workflows/ci-web.yml)
[![Code Quality](https://github.com/autobutler-org/quark/actions/workflows/check.yml/badge.svg)](https://github.com/autobutler-org/quark/actions/workflows/check.yml)
[![CodeQL](https://github.com/autobutler-org/quark/actions/workflows/codeql.yml/badge.svg?branch=main)](https://github.com/autobutler-org/quark/actions/workflows/codeql.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

---

## What is it?

Quark is a self-hosted personal cloud that runs on a device in your home. Think Google Drive or iCloud, except the data never leaves your house and nobody's training AI on your family photos.

You buy the hardware once. No subscriptions. No data mining. It's yours.

## Stack

- **Go** + Gin — backend server
- **Flutter** — cross-platform frontend (web, iOS, Android)
- **SQLite** — embedded local database
- **OpenTelemetry** — observability

## Getting started

**Prerequisites:** Go, Flutter, Make, [air](https://github.com/air-verse/air), sqlc, swag

```bash
git clone https://github.com/autobutler-org/quark.git # Include --recursive if you want to do OS image builds
cd quark
make setup
make generate
make build
```

### Run it locally

Backend with hot reload:

```bash
make watch/backend
```

Frontend (web):

```bash
make serve/frontend
```

Frontend (mobile emulator):

```bash
make emulate          # default platform
make emulate/android  # or emulate/ios
make serve/frontend/mobile
```

> USB device mounting requires root on Linux. Use `AS_ROOT=1` with any backend target if you need it.

Swagger UI is at `http://localhost:8080/swagger` once the backend is running.

### Other useful commands

```bash
make check    # lint
make format   # format
make help     # all targets
```

## Docs

- [Dev Onboarding](docs/dev-onboarding.md)
- [Authentication](docs/auth.md)
- [Mobile Setup](docs/mobile-setup.md)
- [Raspberry Pi Setup](os/README.md)
- [Release Process](docs/release.md)
- [Contributing](CONTRIBUTING.md)

## Contributing

We use linear commit history — one focused commit per PR is the norm. See [CONTRIBUTING.md](CONTRIBUTING.md) for the full rundown.

## License

MIT. See [LICENSE](LICENSE).
