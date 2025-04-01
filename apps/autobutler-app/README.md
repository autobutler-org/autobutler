# autobutler

CLI and UI

FastAPI

## Features

- Access inventory
  - `GET: /api/v1/inv` - Get all inventory
  - `GET: /api/v1/inv?query=str` - Get an inventory item by query
  - `PUT: /api/v1/inv` - Update inventory (called automagically by the butler)
- Access home device
  - `GET: /api/v1/dev?header-only=true` - Get all devices, optionally with headers only
  - `GET: /api/v1/dev/{id}` - Get a device by id
  - `POST: /api/v1/dev` - Add a new device
  - `PUT: /api/v1/dev` - Update a device
  - `DELETE: /api/v1/dev` - Delete a device
