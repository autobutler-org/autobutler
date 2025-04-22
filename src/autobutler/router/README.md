# Chat Service Backend

This is the backend service for the chat application that will integrate with a local LLM.

## Setup

1. Make sure you have Go 1.21 or later installed
2. Install dependencies:
   ```bash
   go mod tidy
   ```


## Running the Service

To run the service:

```bash
go run main.go
```

The server will start on port 8080 by default, or the port specified in your environment variable `PORT`.

## API Endpoints

### POST /api/chat

Send a chat message to the service. This endpoint forwards the request to the LLM server.

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

### POST /api/dummy

Test endpoint that returns a dummy response without calling the LLM.

Request body:

```json
{
  "message": "Your message here"
}
```

Response:

```json
{
  "response": "Hello World! This is a dummy response from the backend. Your message was: Your message here"
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

## Configuration

The service uses the following configuration:
- Server port: Specified via `PORT` environment variable (default: 8080)
- LLM server URL: Specified via `LLM_URL` environment variable or urls.json config
- API endpoints: Configured in urls.json

## API Documentation

The API is documented using the OpenAPI Specification. You can find the specification in the `swagger.yaml` file.

To view the API documentation interactively, you can use Swagger UI:

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
