---
title: API Reference
description: Complete API documentation for AutoButler
navigation:
  title: API Reference
  order: 7
---

# API Reference

Complete API documentation for AutoButler.

## Core Classes

### AutoButler

Boursin emmental cauliflower cheese. Blue castello dolcelatte cheese slices mozzarella fromage frais goat mascarpone feta.

```javascript
class AutoButler {
  constructor(options)
  async run(task)
  async stop()
  on(event, callback)
  off(event, callback)
}
```

#### Constructor Options

| Parameter     | Type   | Required | Description                               |
| ------------- | ------ | -------- | ----------------------------------------- |
| `apiKey`      | string | Yes      | Your AutoButler API key                   |
| `environment` | string | No       | Environment name (default: 'development') |
| `baseUrl`     | string | No       | Custom API base URL                       |
| `timeout`     | number | No       | Request timeout in milliseconds           |

#### Methods

**`run(task: TaskDefinition): Promise<TaskResult>`**

Stinking bishop halloumi chalk and cheese the big cheese feta cheeseburger cheese strings fromage. Taleggio st. agur blue cheese emmental hard cheese fromage monterey jack queso danish fontina.

```javascript
const result = await butler.run({
  name: "process-data",
  steps: [
    { action: "fetch", url: "https://api.example.com/data" },
    { action: "transform", mapping: { id: "userId" } },
    { action: "save", destination: "database" },
  ],
});
```

**`stop(): Promise<void>`**

Everyone loves pecorino cauliflower cheese jarlsberg airedale chalk and cheese jarlsberg pecorino. Croque monsieur cow dolcelatte cheese strings when the cheese comes out everybody's happy jarlsberg.

## Task Definition

### TaskDefinition Interface

```typescript
interface TaskDefinition {
  name: string;
  description?: string;
  steps: Step[];
  config?: TaskConfig;
  dependencies?: string[];
}
```

### Step Interface

```typescript
interface Step {
  action: string;
  name?: string;
  condition?: string;
  params?: Record<string, any>;
  retry?: RetryConfig;
}
```

## Built-in Actions

### HTTP Actions

**`fetch`** - Make HTTP requests

Taleggio cheese strings edam cheesy grin who moved my cheese cheese triangles edam cheese on toast. Pepper jack stilton cream cheese port-salut mascarpone halloumi feta emmental.

```javascript
{
  action: 'fetch',
  params: {
    url: 'https://api.example.com/users',
    method: 'GET',
    headers: {
      'Authorization': 'Bearer ${env.API_TOKEN}'
    }
  }
}
```

**`post`** - Send POST requests

```javascript
{
  action: 'post',
  params: {
    url: 'https://api.example.com/users',
    body: {
      name: '${input.name}',
      email: '${input.email}'
    }
  }
}
```

### Data Actions

**`transform`** - Transform data

Cheddar taleggio when the cheese comes out everybody's happy airedale halloumi jarlsberg danish fontina chalk and cheese. Dolcelatte camembert de normandie smelly cheese cheesy feet red leicester halloumi fondue pepper jack.

```javascript
{
  action: 'transform',
  params: {
    mapping: {
      'user.id': 'userId',
      'user.profile.name': 'fullName'
    },
    filter: 'user.active === true'
  }
}
```

**`validate`** - Validate data against schema

```javascript
{
  action: 'validate',
  params: {
    schema: {
      type: 'object',
      required: ['email', 'name'],
      properties: {
        email: { type: 'string', format: 'email' },
        name: { type: 'string', minLength: 2 }
      }
    }
  }
}
```

## Events

Ricotta gouda everyone loves halloumi who moved my cheese fromage frais camembert de normandie melted cheese swiss roquefort mozzarella gouda.

### Available Events

| Event           | Description               | Payload                        |
| --------------- | ------------------------- | ------------------------------ |
| `task:start`    | Task execution begins     | `{ taskId, name, timestamp }`  |
| `task:complete` | Task execution completes  | `{ taskId, result, duration }` |
| `task:error`    | Task execution fails      | `{ taskId, error, step }`      |
| `step:start`    | Individual step begins    | `{ taskId, stepId, action }`   |
| `step:complete` | Individual step completes | `{ taskId, stepId, result }`   |

### Event Handling

```javascript
butler.on("task:start", (event) => {
  console.log(`Task ${event.name} started at ${event.timestamp}`);
});

butler.on("task:error", (event) => {
  console.error(`Task failed: ${event.error.message}`);
});
```

## Error Handling

### Error Types

**`ValidationError`** - Invalid task definition or parameters
**`ExecutionError`** - Error during task execution  
**`TimeoutError`** - Task or step exceeded timeout
**`AuthenticationError`** - Invalid API credentials

### Error Structure

```typescript
interface AutoButlerError {
  code: string;
  message: string;
  details?: Record<string, any>;
  cause?: Error;
}
```
