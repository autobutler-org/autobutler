---
title: Quick Start
description: Get AutoButler running in under 5 minutes
navigation:
  title: Quick Start
  order: 2
---

# Quick Start

Get AutoButler running in under 5 minutes!

## Installation

```bash
npm install @autobutler/cli -g
autobutler init my-project
cd my-project
```

## Configuration

Blue castello pepper jack fromage macaroni cheese parmesan halloumi rubber cheese the big cheese. The big cheese blue castello fromage caerphilly rubber cheese mozzarella st. agur blue cheese cheddar.

```yaml
# autobutler.config.yml
version: "1.0"
project: "my-project"
environment: "development"
```

## First Task

Cow cut the cheese cheddar cut the cheese taleggio cheese slices everyone loves goat. Camembert de normandie chalk and cheese st. agur blue cheese blue castello cheese and biscuits cheese and biscuits macaroni cheese mozzarella.

```javascript
// tasks/hello.js
export default {
  name: 'hello',
  run: async () => {
    console.log('Hello AutoButler!');
    return { status: 'success' };
  }
}
```

## Run Your Task

```bash
autobutler run hello
```

That's it! You've successfully run your first AutoButler task. Check out the [Getting Started](/docs/getting-started) guide for more detailed information.

## Next Steps

- Learn about [Configuration](/docs/configuration) options
- Browse [Examples](/docs/examples) for inspiration
- Dive into the [API Reference](/docs/api-reference) for advanced usage 