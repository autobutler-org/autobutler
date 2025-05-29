---
title: Installation
description: Detailed installation instructions for AutoButler
navigation:
  title: Installation
  order: 4
---

# Installation

Complete installation guide for AutoButler on all supported platforms.

## Prerequisites

Before installing AutoButler, ensure you have the following prerequisites:

- Node.js 18.0 or higher
- npm 9.0 or higher (or yarn/pnpm equivalent)
- Git for version control

## Quick Install

The fastest way to get started with AutoButler:

```bash
npm install -g @autobutler/cli
autobutler init my-project
cd my-project
npm start
```

## Platform-Specific Installation

### Windows Installation

For Windows users, we recommend using PowerShell or Windows Terminal:

1. Install Node.js from [nodejs.org](https://nodejs.org)
2. Open PowerShell as Administrator
3. Run the installation command:

```powershell
npm install -g @autobutler/cli
```

### macOS Installation

For macOS users, you can use Homebrew or install directly:

#### Using Homebrew

```bash
brew install node
npm install -g @autobutler/cli
```

#### Direct Installation

```bash
# Install AutoButler CLI
npm install -g @autobutler/cli

# Verify installation
autobutler --version
```

### Linux Installation

For Linux distributions, use your package manager or install directly:

#### Ubuntu/Debian

```bash
# Update package index
sudo apt update

# Install Node.js and npm
sudo apt install nodejs npm

# Install AutoButler CLI
npm install -g @autobutler/cli
```

#### CentOS/RHEL/Fedora

```bash
# Install Node.js and npm
sudo yum install nodejs npm

# Install AutoButler CLI
npm install -g @autobutler/cli
```

## Verification

After installation, verify that AutoButler is working correctly:

```bash
# Check version
autobutler --version

# Run help command
autobutler --help

# Create a test project
autobutler init test-project
```

## Troubleshooting

### Common Issues

If you encounter issues during installation:

1. **Permission Errors**: Use `sudo` on Unix systems or run as Administrator on Windows
2. **Network Issues**: Check your internet connection and proxy settings
3. **Node Version**: Ensure you're using Node.js 18.0 or higher

### Getting Help

If you need assistance:

- Check our [FAQ](/docs/help)
- Visit our [GitHub Issues](https://github.com/autobutler/autobutler/issues)
- Join our [Discord Community](https://discord.gg/autobutler)