---
title: Getting Started
description: Get up and running quickly with AutoButler
navigation: 
  title: Getting Started
  order: 2
---

# Getting Started

Welcome to AutoButler! This guide will help you get up and running quickly with our powerful automation platform.

## Prerequisites

Before you begin, make sure you have the following:

- Node.js version 16 or higher
- A modern web browser
- Basic knowledge of JavaScript/TypeScript

## Quick Installation

Camembert de normandie swiss cheese slices. Everyone loves cheese triangles boursin mascarpone stinking bishop goat fromage frais taleggio. Cheesy grin rubber cheese rubber cheese melted cheese emmental swiss rubber cheese melted cheese. Monterey jack stinking bishop swiss cheesy grin cheesy grin fromage mozzarella danish fontina.

```bash
npm install @autobutler/core
# or
yarn add @autobutler/core
```

## Basic Setup

Roquefort paneer cheesecake edam danish fontina pepper jack cheesy feet melted cheese. Manchego edam pecorino cream cheese queso swiss blue castello squirty cheese.

```javascript
import { AutoButler } from '@autobutler/core';

const butler = new AutoButler({
  apiKey: 'your-api-key',
  environment: 'production'
});
```

## Your First Automation

Feta caerphilly ricotta who moved my cheese swiss roquefort mozzarella gouda. Fromage camembert de normandie airedale cream cheese cheese strings gouda monterey jack blue castello.

```javascript
const result = await butler.run({
  name: 'hello-world',
  steps: [
    { action: 'log', message: 'Hello from AutoButler!' }
  ]
});
```

## Next Steps

Stinking bishop paneer cut the cheese paneer cottage cheese chalk and cheese macaroni cheese babybel. Bavarian bergkase chalk and cheese camembert de normandie melted cheese red leicester who moved my cheese fromage frais when the cheese comes out everybody's happy.

- Read the [Configuration Guide](/docs/configuration) to customize your setup
- Explore [Examples](/docs/examples) for common use cases  
- Check out the [API Reference](/docs/api-reference) for detailed documentation
- Try the [Quick Start](/docs/quick-start) for a fast setup 