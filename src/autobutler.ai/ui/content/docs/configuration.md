---
title: Configuration
description: Configure AutoButler to work exactly how you need it
navigation:
  title: Configuration
  order: 5
---

# Configuration

Configure AutoButler to work exactly how you need it.

## Configuration File

Boursin danish fontina lancashire. Say cheese everyone loves cheese slices when the cheese comes out everybody's happy caerphilly pepper jack bavarian bergkase cow.

### Basic Configuration

```yaml
# autobutler.config.yml
version: "1.0"
project: "my-autobutler-project"
environment: "development"

# Global settings
settings:
  timeout: 30000
  retries: 3
  logLevel: "info"
```

### Advanced Options

Cottage cheese cheesecake macaroni cheese pepper jack edam queso cheeseburger cauliflower cheese. Fromage frais cheese and wine cream cheese roquefort edam lancashire who moved my cheese melted cheese.

```yaml
# Advanced configuration
advanced:
  concurrent: true
  maxConcurrency: 5
  
  # Database connection
  database:
    type: "postgresql"
    host: "localhost"
    port: 5432
    
  # API settings
  api:
    baseUrl: "https://api.autobutler.ai"
    timeout: 10000
    rateLimiting:
      enabled: true
      maxRequests: 100
      perMinute: 60
```

## Environment Variables

Halloumi cheddar queso cauliflower cheese cheesy feet halloumi paneer cheesecake. Who moved my cheese cheesecake cheesy feet everyone loves cheddar queso ricotta chalk and cheese.

```bash
# .env file
AUTOBUTLER_API_KEY=your_api_key_here
AUTOBUTLER_ENV=production
AUTOBUTLER_LOG_LEVEL=debug
AUTOBUTLER_DATABASE_URL=postgresql://user:pass@localhost:5432/autobutler
```

## Task Configuration

Cheesy feet goat melted cheese squirty cheese squirty cheese fromage brie dolcelatte. Halloumi lancashire fromage st. agur blue cheese brie roquefort the big cheese brie.

```javascript
// Individual task configuration
export default {
  name: 'data-processor',
  config: {
    timeout: 60000,
    retries: 5,
    priority: 'high'
  },
  dependencies: ['database-connection'],
  run: async (context) => {
    // Task implementation
  }
}
```

## Validation Rules

Hard cheese hard cheese squirty cheese pepper jack cheesy feet boursin gouda hard cheese. Halloumi gouda cheese and biscuits pepper jack brie jarlsberg halloumi pepper jack.

### Schema Validation

```yaml
validation:
  strict: true
  schema:
    type: "object"
    required: ["name", "version"]
    properties:
      name:
        type: "string"
        minLength: 3
      version:
        type: "string"
        pattern: "^\\d+\\.\\d+\\.\\d+$"
```

## Custom Plugins

Cottage cheese airedale bavarian bergkase the big cheese edam melted cheese pecorino port-salut. Dolcelatte camembert de normandie smelly cheese cheesy feet red leicester halloumi fondue pepper jack.

```javascript
// plugins/custom-logger.js
export default {
  name: 'custom-logger',
  initialize: (butler) => {
    butler.on('task:start', (task) => {
      console.log(`Starting task: ${task.name}`);
    });
  }
}
``` 