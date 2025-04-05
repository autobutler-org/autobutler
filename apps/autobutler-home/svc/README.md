# Chat Service Backend

This is the backend service for the chat application that will integrate with a local LLM.

## Setup

1. Make sure you have Go 1.21 or later installed
2. Install dependencies:
   ```bash
   go mod tidy
   ```
3. Create a `.env` file (optional):
   ```
   PORT=8080
   ```

## Running the Service

To run the service:
```bash
go run main.go
```

The server will start on port 8080 by default, or the port specified in your `.env` file.

## API Endpoints

### POST /api/chat
Send a chat message to the service.

Request body:
```json
{
    "message": "Your message here"
}
```

Response:
```json
{
    "response": "Service response"
}
```

### GET /health
Health check endpoint.

Response:
```json
{
    "status": "ok"
}
```

## Development

The service is built with:
- Gin web framework
- Environment variable support
- CORS enabled for frontend integration

## TODO
- [ ] Integrate with local LLM
- [ ] Add authentication
- [ ] Add rate limiting
- [ ] Add request validation
- [ ] Add proper error handling 