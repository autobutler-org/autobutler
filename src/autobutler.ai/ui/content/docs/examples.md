---
title: Examples
description: Real-world examples to help you get the most out of AutoButler
navigation:
  title: Examples
  order: 5
---

# Examples

Real-world examples to help you get the most out of AutoButler.

## Basic Examples

### Simple Data Fetching

Cheese strings when the cheese comes out everybody's happy jarlsberg. Taleggio cheese strings edam cheesy grin who moved my cheese cheese triangles edam cheese on toast.

```javascript
const dataFetcher = {
  name: 'fetch-user-data',
  description: 'Fetch user data from API',
  steps: [
    {
      action: 'fetch',
      name: 'get-users',
      params: {
        url: 'https://jsonplaceholder.typicode.com/users',
        method: 'GET'
      }
    },
    {
      action: 'transform',
      name: 'format-users',
      params: {
        mapping: {
          'name': 'fullName',
          'email': 'emailAddress',
          'address.city': 'location'
        }
      }
    }
  ]
};

await butler.run(dataFetcher);
```

### File Processing

Pepper jack stilton cream cheese port-salut mascarpone halloumi feta emmental. Cheddar taleggio when the cheese comes out everybody's happy airedale halloumi jarlsberg danish fontina chalk and cheese.

```javascript
const fileProcessor = {
  name: 'process-csv-file',
  steps: [
    {
      action: 'read-file',
      params: {
        path: './data/users.csv',
        format: 'csv'
      }
    },
    {
      action: 'validate',
      params: {
        schema: {
          type: 'array',
          items: {
            type: 'object',
            required: ['email', 'name']
          }
        }
      }
    },
    {
      action: 'transform',
      params: {
        filter: 'item.active === true',
        mapping: {
          'email': 'email',
          'name': 'displayName'
        }
      }
    },
    {
      action: 'write-file',
      params: {
        path: './output/processed-users.json',
        format: 'json'
      }
    }
  ]
};
```

## Advanced Examples

### API Integration with Error Handling

Dolcelatte camembert de normandie smelly cheese cheesy feet red leicester halloumi fondue pepper jack. Ricotta gouda everyone loves halloumi who moved my cheese fromage frais camembert de normandie.

```javascript
const apiIntegration = {
  name: 'sync-user-data',
  description: 'Sync user data between systems',
  config: {
    timeout: 30000,
    retries: 3
  },
  steps: [
    {
      action: 'fetch',
      name: 'get-source-users',
      params: {
        url: '${env.SOURCE_API}/users',
        headers: {
          'Authorization': 'Bearer ${env.SOURCE_TOKEN}'
        }
      },
      retry: {
        attempts: 5,
        delay: 2000,
        backoff: 'exponential'
      }
    },
    {
      action: 'foreach',
      name: 'process-each-user',
      params: {
        items: '${steps.get-source-users.result}',
        parallel: true,
        maxConcurrency: 5
      },
      steps: [
        {
          action: 'condition',
          params: {
            if: 'item.status === "active"'
          },
          steps: [
            {
              action: 'post',
              params: {
                url: '${env.TARGET_API}/users',
                body: {
                  id: '${item.id}',
                  name: '${item.name}',
                  email: '${item.email}',
                  lastSync: '${Date.now()}'
                }
              }
            }
          ]
        }
      ]
    }
  ]
};
```

### Database Operations

```javascript
const databaseSync = {
  name: 'database-sync',
  dependencies: ['database-connection'],
  steps: [
    {
      action: 'query',
      name: 'get-updated-records',
      params: {
        sql: `
          SELECT id, name, email, updated_at 
          FROM users 
          WHERE updated_at > ?
        `,
        params: ['${input.lastSyncTime}']
      }
    },
    {
      action: 'foreach',
      name: 'sync-records',
      params: {
        items: '${steps.get-updated-records.result}'
      },
      steps: [
        {
          action: 'upsert',
          params: {
            table: 'user_cache',
            data: {
              user_id: '${item.id}',
              name: '${item.name}',
              email: '${item.email}',
              last_updated: '${item.updated_at}'
            },
            conflictColumns: ['user_id']
          }
        }
      ]
    }
  ]
};
```

## Workflow Examples

### Multi-Step Data Pipeline

Melted cheese swiss roquefort mozzarella gouda cheese and wine danish fontina cheese and wine boursin when the cheese comes out everybody's happy mozzarella lancashire cheese and biscuits.

```javascript
const dataPipeline = {
  name: 'customer-analytics-pipeline',
  description: 'Process customer data for analytics',
  steps: [
    // Extract
    {
      action: 'parallel',
      name: 'extract-data',
      steps: [
        {
          action: 'fetch',
          name: 'get-customers',
          params: {
            url: '${env.CRM_API}/customers',
            headers: { 'API-Key': '${env.CRM_KEY}' }
          }
        },
        {
          action: 'fetch',
          name: 'get-orders',
          params: {
            url: '${env.ECOMMERCE_API}/orders',
            headers: { 'Authorization': 'Bearer ${env.ECOMMERCE_TOKEN}' }
          }
        },
        {
          action: 'query',
          name: 'get-support-tickets',
          params: {
            sql: 'SELECT * FROM support_tickets WHERE created_at >= ?',
            params: ['${input.startDate}']
          }
        }
      ]
    },
    
    // Transform
    {
      action: 'transform',
      name: 'merge-customer-data',
      params: {
        script: `
          const customers = steps['extract-data'].results['get-customers'];
          const orders = steps['extract-data'].results['get-orders'];
          const tickets = steps['extract-data'].results['get-support-tickets'];
          
          return customers.map(customer => ({
            ...customer,
            totalOrders: orders.filter(o => o.customerId === customer.id).length,
            totalSpent: orders
              .filter(o => o.customerId === customer.id)
              .reduce((sum, o) => sum + o.amount, 0),
            supportTickets: tickets.filter(t => t.customerId === customer.id).length
          }));
        `
      }
    },
    
    // Load
    {
      action: 'batch-insert',
      name: 'save-analytics',
      params: {
        table: 'customer_analytics',
        data: '${steps.merge-customer-data.result}',
        batchSize: 1000
      }
    }
  ]
};
```

### Notification System

Cow boursin smelly cheese cheese and biscuits emmental cheesy feet. Ricotta caerphilly when the cheese comes out everybody's happy halloumi cheese and biscuits cheesy feet cheesecake fondue.

```javascript
const notificationSystem = {
  name: 'alert-system',
  description: 'Monitor and send alerts',
  config: {
    schedule: '*/5 * * * *' // Every 5 minutes
  },
  steps: [
    {
      action: 'query',
      name: 'check-system-health',
      params: {
        sql: `
          SELECT service_name, status, last_check 
          FROM service_health 
          WHERE status != 'healthy'
        `
      }
    },
    {
      action: 'condition',
      params: {
        if: 'steps["check-system-health"].result.length > 0'
      },
      steps: [
        {
          action: 'foreach',
          name: 'send-alerts',
          params: {
            items: '${steps.check-system-health.result}'
          },
          steps: [
            {
              action: 'post',
              name: 'slack-notification',
              params: {
                url: '${env.SLACK_WEBHOOK}',
                body: {
                  text: `🚨 Service Alert: ${item.service_name} is ${item.status}`,
                  channel: '#alerts'
                }
              }
            },
            {
              action: 'post',
              name: 'email-notification',
              params: {
                url: '${env.EMAIL_API}/send',
                body: {
                  to: '${env.ADMIN_EMAIL}',
                  subject: 'Service Alert',
                  body: `Service ${item.service_name} requires attention. Status: ${item.status}`
                }
              }
            }
          ]
        }
      ]
    }
  ]
};
``` 