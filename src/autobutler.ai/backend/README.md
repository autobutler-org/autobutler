# Customer API Documentation

## Overview

This is the customer-facing API for our SaaS platform. It handles authentication, purchases, and customer support for our $359 base product.

**Base URL:** `https://customers.yoursite.com/api/v1`

## Business Model

- **Base Product**: $359 one-time purchase

## Authentication

We use **Auth0** for authentication with JWT bearer tokens.

### Auth Flow

1. User logs in through Auth0
2. Auth0 redirects to `/auth/callback` with authorization code
3. API exchanges code for JWT tokens
4. Client uses bearer token for subsequent requests

### Endpoints

- `POST /auth/callback` - Exchange Auth0 code for tokens
- `POST /auth/refresh` - Refresh access token
- `POST /auth/logout` - Logout user

## Core Features

### Customer Profile Management

- `GET /profile` - Get customer profile
- `PUT /profile` - Update profile (name, phone, company)

### Purchasing & Checkout

- `POST /checkout/create-session` - Create Stripe checkout session
- `POST /checkout/success` - Process successful payment

### Purchase History

- `GET /purchases` - List all purchases (paginated)
- `GET /purchases/{id}` - Get specific purchase details

### Billing & Payments

- `GET /billing/payment-methods` - List saved payment methods
- `POST /billing/payment-methods` - Add new payment method
- `DELETE /billing/payment-methods/{id}` - Remove payment method
- `GET /billing/invoices` - List invoices and receipts

### License Management

- `GET /licenses` - Get license keys for purchased products

### Downloads

- `GET /downloads` - List available software downloads
- `GET /downloads/{id}/url` - Get signed download URL (expires after time limit)

### Customer Support

- `GET /support/tickets` - List support tickets
- `POST /support/tickets` - Create new support ticket
- `GET /support/tickets/{id}` - Get ticket details with message history
- `POST /support/tickets/{id}/messages` - Add message to ticket

## Product Types

The API handles one product type:

- `base_product` - The main $359 software

## Key Data Models

### Purchase

```json
{
  "id": "purchase_123",
  "amount": 35900,
  "currency": "usd",
  "status": "completed",
  "items": [
    {
      "productType": "base_product",
      "description": "Base Software License",
      "amount": 35900,
      "quantity": 1
    }
  ],
  "createdAt": "2025-01-15T10:30:00Z"
}
```

### License

```json
{
  "id": "license_789",
  "key": "XXXX-XXXX-XXXX-XXXX",
  "productType": "base_product",
  "status": "active",
  "expiresAt": null
}
```

## Integration Points

### Stripe Integration

- Checkout sessions for one-time purchases
- Payment method management
- Invoice generation

### Auth0 Integration

- User authentication and registration
- JWT token management
- User profile synchronization

### Download Security

- Signed URLs with expiration
- License validation before download access
- Version-based access control

## Usage Examples

### Creating a Purchase

```bash
# 1. Create checkout session
curl -X POST /checkout/create-session \
  -H "Authorization: Bearer {token}" \
  -d '{
    "items": [{"productType": "base_product"}],
    "successUrl": "https://yoursite.com/success",
    "cancelUrl": "https://yoursite.com/cancel"
  }'

# 2. Redirect user to Stripe checkout
# 3. Handle success callback
curl -X POST /checkout/success \
  -H "Authorization: Bearer {token}" \
  -d '{"sessionId": "cs_xxx"}'
```

### Getting Download Access

```bash
# 1. Check available downloads
curl -H "Authorization: Bearer {token}" /downloads

# 2. Get signed download URL
curl -H "Authorization: Bearer {token}" /downloads/software_v2/url
```

### Creating Support Ticket

```bash
curl -X POST /support/tickets \
  -H "Authorization: Bearer {token}" \
  -d '{
    "subject": "Installation Issue",
    "description": "Cannot install on Windows 11",
    "category": "technical",
    "priority": "medium"
  }'
```

## Error Handling

All endpoints return standard HTTP status codes:

- `200` - Success
- `201` - Created
- `400` - Bad Request
- `401` - Unauthorized
- `403` - Forbidden
- `404` - Not Found
- `500` - Internal Server Error

## Rate Limiting

API requests are rate limited per user:

- 100 requests per minute for most endpoints
- 10 requests per minute for checkout endpoints

## Next Steps

1. Import the Swagger spec into Postman
2. Set up Auth0 configuration
3. Configure Stripe webhooks
4. Implement download file storage
5. Set up support ticket notifications

## Related APIs

This customer API works alongside our internal business operations API (separate subdomain) that handles:

- Admin customer management
- Financial reporting
- Product management
- Support queue management
