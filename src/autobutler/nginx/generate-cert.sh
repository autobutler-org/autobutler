#!/bin/bash

# Create directory for SSL certificates if it doesn't exist
mkdir -p ssl

# Generate a private key
openssl genrsa -out ssl/server.key 2048

# Generate a certificate signing request (CSR)
openssl req -new -key ssl/server.key -out ssl/server.csr -subj "/C=US/ST=State/L=City/O=Organization/CN=localhost"

# Generate a self-signed certificate valid for 365 days
openssl x509 -req -days 365 -in ssl/server.csr -signkey ssl/server.key -out ssl/server.crt

# Set appropriate permissions
chmod 600 ssl/server.key
chmod 644 ssl/server.crt

echo "Self-signed SSL certificate has been generated"
echo "Certificate: ssl/server.crt"
echo "Private Key: ssl/server.key" 