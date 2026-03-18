# AutoButler

Your own private cloud, running in your house. Photos, files, documents — all on hardware you own, off servers you don't trust.

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

---

## What is it?

AutoButler is a self-hosted personal cloud that runs on a device in your home. Think Google Drive or iCloud, except the data never leaves your house and nobody's training AI on your family photos.

You buy the hardware once. No subscriptions. No data mining. It's yours.

## Stack

- **Go** + Gin — backend server
- **Flutter** — cross-platform frontend (web, iOS, Android)
- **SQLite** — embedded local database
- **OpenTelemetry** — observability

## Getting started

**Prerequisites:** Go, Flutter, Make, [air](https://github.com/air-verse/air), sqlc, swag

```bash
git clone https://github.com/autobutler-org/autobutler.git
cd autobutler
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

- [Authentication](docs/auth.md)
- [Contributing](CONTRIBUTING.md)
- [Azure Deployment](docs/azure-deployment.md)

## Contributing

We use linear commit history — one focused commit per PR is the norm. See [CONTRIBUTING.md](CONTRIBUTING.md) for the full rundown.

## License

MIT. See [LICENSE](LICENSE).
