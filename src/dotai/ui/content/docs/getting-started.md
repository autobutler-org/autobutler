---
title: Getting Started
description: Get up and running quickly with AutoButler
navigation:
  title: Getting Started
  order: 2
---

# Getting Started

Welcome to AutoButler! This guide will help you get up and running quickly with our platform.

## Prerequisites

Before you begin, make sure you have the following:

- A modern web browser
- A basic level of comfort running commands in a terminal

## Quick Installation

To install autobutler, you simply need to download the binary for your hardware from
[our releases page](https://github.com/autobutler-ai/autobutler.ai/releases).

Once downloaded, you can extract the binary and run the `install` command with your
API key, and `sudo` privileges.

```bash
API_KEY=your-api-key
sudo /path/to/autobutler install $API_KEY
```

If you wish to install it via `curl`, you can run the following command:

```bash
API_KEY=your-api-key
VERSION=v0.0.0
curl \
  --fail \
  -L "https://github.com/exokomodo/autobutler.ai/releases/download/${VERSION}/autobutler_darwin_arm64.tar.gz" | tar -xv
sudo ./autobutler install $API_KEY
```

or if you prefer to use `wget`, you can run:

```bash
API_KEY=your-api-key
VERSION=v0.0.0
wget \
  -qO- \
  "https://github.com/exokomodo/autobutler.ai/releases/download/${VERSION}/autobutler_darwin_arm64.tar.gz" | tar -xv
sudo ./autobutler install $API_KEY
```

Autobutler should now be running in the background, and you can check it out at [http://localhost:8081](http://localhost:8081).

## Next Steps

- Check out the [Help & Support](/docs/help) section for troubleshooting and community resources
