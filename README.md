# AutoButler

> **The New Era of Digital Independence Starts With You**  
> Own your data. Control your future.

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

**NOTE: This product will ship in February 2026.**

---

## What is AutoButler?

AutoButler is a plug-and-play private cloud system that puts you back in control of your digital life. Instead of renting
storage and services from Big Tech, AutoButler is a physical device you own that runs in your home—giving you complete
sovereignty over your data, documents, photos, and digital memories.

The internet used to be a platform that we used, but increasingly has become a data-mining platform for tech companies.
AutoButler seeks to restore control to consumers by storing your data in a physical device in your home, not on someone
else's servers.

### Why AutoButler?

**Own, Don't Rent**  
Remember when buying software meant you owned it? AutoButler brings that philosophy back. No subscriptions for your own
data—just a one-time purchase with optional upgrades when _you_ want them.

**Privacy First**  
Tech giants use your photos for AI training and build shadow profiles of your children. AutoButler keeps your family's
data private, secure, and out of the hands of corporations.

**Physical Ownership**  
Your data lives in a fireproof, waterproof container in your home. Add external drives, mail backups to family, or upgrade
storage—it's yours to manage as you please.

### What You Get With AutoButler

✅ **Your Own Private Cloud** - Stored physically in your home  
✅ **Document Editing & Spreadsheets** - Own it forever, no subscriptions  
✅ **File Storage & Photo Backup** - Automatic, private, secure  
✅ **Fireproof, Waterproof Container** - Military-grade protection  
✅ **Expandable Storage** - Add external drives as needed  
✅ **Complete Data Ownership** - Do what you please with your data

### Who Is AutoButler For?

- **Parents** - Protect your children from shadow profiling and AI data harvesting
- **Privacy Advocates** - Fight for digital sovereignty without sacrificing convenience
- **Anti-Subscription Consumers** - Stop renting your digital life and start owning it
- **Anyone** who believes their data belongs to them, not corporations

---

## What AutoButler Is Made Of

AutoButler is built as a full-stack web application designed to run on dedicated hardware in your home.

### Technology Stack

#### Backend

- **Go 1.24+** - High-performance, compiled backend server
- **Gin Web Framework** - Fast HTTP routing and middleware
- **SQLite** (modernc.org/sqlite) - Embedded database for local data storage
- **golang-migrate** - Database schema migrations
- **templ** - Type-safe HTML templating for Go

#### Frontend

- **Vanilla JavaScript** - No heavy frameworks, just fast, clean JS
- **Tailwind CSS** - Utility-first CSS framework
- **Flowbite Icons** - Beautiful SVG icon library

#### Testing & Quality

- **Playwright** - End-to-end testing framework
- **ESLint & Prettier** - Code quality and formatting
- **Go standard testing** - Unit and integration tests

#### Observability

- **OpenTelemetry** - Instrumentation for monitoring and tracing
- **Runtime metrics** - Performance monitoring

### Key Features Implemented

- **Document Management** - View and organize documents (DOCX support with custom parser)
- **Photo Library** - Browse, organize, and automatically backup photos with EXIF data support
- **File Storage** - Cross-platform device detection and file browsing
- **Device Management** - Track and manage connected storage devices
- **Health Monitoring** - System health checks and status reporting

### Architecture

```plaintext
autobutler/
├── cmd/autobutler/          # CLI entry points (serve, install, version)
├── internal/
│   ├── server/              # HTTP server, routes, middleware
│   │   ├── api/v1/          # REST API endpoints
│   │   ├── ui/              # HTML UI handlers and components
│   │   └── public/          # Static assets (CSS, JS, images)
│   └── install/             # Installation and service management
├── pkg/
│   ├── calendar/            # Calendar domain logic
│   ├── db/                  # Database layer with sqlc-generated code
│   ├── storage/             # Cross-platform storage detection
│   └── util/                # Shared utilities
├── sql/queries/             # SQL queries for code generation
└── tests/e2e/               # End-to-end Playwright tests
```

---

## Getting Started

### Prerequisites

- **Go 1.24+** - [Install Go](https://golang.org/doc/install)
- **Make** - Build automation
- **Node.js & npm** - For frontend tooling and tests
- **air** - Hot reloading tool
- **sqlc** - SQL code generator (for database layer development)
- **swag** - Generator for Swagger documentation

### Installation

1. **Clone the repository**

   ```bash
   git clone https://github.com/autobutler-org/autobutler.git
   cd autobutler
   ```

2. **Set up development environment**

   ```bash
   make setup
   ```

3. **Generate code** (templ templates, database code, etc.)

   ```bash
   make generate
   ```

4. **Build the application**

   ```bash
   make build
   ```

### Development

#### Run the frontend and backend with hot-reloading

```bash
make watch
```

#### Run the frontend and backend (production mode)

```bash
make serve
```

#### Access the Swagger UI

Go to [`http://localhost:8080/swagger`](http://localhost:8080/swagger)

### Root Privileges

**Note:** Mounting and unmounting USB storage devices requires root privileges on Linux. For development and testing, you
must run the backend as root to perform these operations. The Makefile supports this with the `AS_ROOT=1` environment variable:

```bash
make watch/backend AS_ROOT=1
```

or

```bash
make serve/backend AS_ROOT=1
```

This ensures the backend process has the necessary permissions to execute mount and unmount commands. Without root, device
mounting and unmounting will fail with permission errors.

#### Run end-to-end tests

```bash
npm run test/e2e
```

#### View test reports

```bash
npm run test/e2e/report
```

#### Format and lint code

```bash
make format        # Format all code
```

#### Project Commands

View all available commands:

```bash
make help
```

---

## Contributing

We welcome contributions! AutoButler is built on the principle of digital sovereignty, and we believe in transparent,
community-driven development.

Before contributing, please:

1. Read our [Contributing Guide](CONTRIBUTING.md)
2. Review the existing [issues](https://github.com/autobutler-org/autobutler/issues)
3. Sign your commits for authenticity
4. Keep commits focused and atomic (we use linear history)

### Quick Contributing Tips

- **Code Style**: Follow existing conventions; use `make fmt` before committing
- **Commit Messages**: Clear, concise, under 80 characters
- **Pull Requests**: One focused change per PR
- **Testing**: Add tests for new features; ensure existing tests pass

See [CONTRIBUTING.md](CONTRIBUTING.md) for complete guidelines.

---

## Documentation

- [Contributing Guide](CONTRIBUTING.md) - How to contribute to the project
- [Azure Deployment](docs/azure-deployment.md) - Deploy AutoButler to Azure
- [EPUB Documentation](docs/epub/) - EPUB file format documentation
- [Source Code](https://github.com/autobutler-org/autobutler) - GitHub repository

---

## Important Links

- [GitHub Repository](https://github.com/autobutler-org/autobutler)
- [Issue Tracker](https://github.com/autobutler-org/autobutler/issues)
- [SVG Icon Library](https://flowbite.com/icons/)

---

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## Philosophy

The future is about the people of the internet, not the corporations. We stand for digital sovereignty.

**What if your online services worked more like plumbers than subscriptions?**

Instead of renting your device and storage, you own your own private cloud. Pay for fixes or upgrades... or do them yourself.
It's yours to manage as you please.

Stop renting your digital life. **Start owning it.**
