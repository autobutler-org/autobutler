#!/bin/bash

echo "Setting up HTTPS for AutoButler..."

# Create nginx directory if it doesn't exist
mkdir -p nginx/ssl

# Generate self-signed certificates
cd nginx/ssl || exit 1
echo "Generating SSL certificates..."

# Generate private key
openssl genrsa -out server.key 2048

# Generate self-signed certificate
openssl req -new -x509 -key server.key -out server.crt -days 365 -subj "/CN=localhost" -addext "subjectAltName = DNS:localhost"

# Set permissions
chmod 600 server.key
chmod 644 server.crt

cd ../.. || exit 1

echo "Starting the services..."
podman compose down
podman compose up