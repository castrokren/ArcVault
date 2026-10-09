#!/bin/bash

# ArcVault Docker Deployment Script for macOS/Linux
# Deploys ArcVault server using Docker for cross-platform support

set -e

echo "========================================"
echo "ArcVault Docker Deployment"
echo "========================================"
echo

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo "ERROR: Docker is not installed."
    echo "Please install Docker and try again."
    exit 1
fi

# Check if Docker Compose is installed
if ! command -v docker-compose &> /dev/null; then
    echo "ERROR: Docker Compose is not installed."
    echo "Please install Docker Compose and try again."
    exit 1
fi

# Create data directories
echo "Creating data directories..."
mkdir -p data/keys data/logs data/config nginx

# Copy configuration templates
echo "Copying configuration templates..."
cp env.templates/production.env data/config/production.env
cp env.templates/staging.env data/config/staging.env
cp env.templates/development.env data/config/development.env

# Generate self-signed certificates if they don't exist
if [ ! -f "data/keys/server.crt" ] || [ ! -f "data/keys/server.key" ]; then
    echo "Generating self-signed certificates..."
    
    # Check if we have the certificate generator
    if [ -f "scripts/generate_self_signed_cert.py" ]; then
        python3 scripts/generate_self_signed_cert.py data/keys localhost
    else
        echo "WARNING: Certificate generator not found."
        echo "You may need to generate certificates manually."
    fi
fi

# Generate JWT keys if they don't exist
if [ ! -f "data/keys/controller_private.pem" ] || [ ! -f "data/keys/controller_public.pem" ]; then
    echo "Generating JWT keys..."
    
    # Check if we have the JWT key generator
    if [ -f "scripts/generate_jwt_keys.py" ]; then
        python3 scripts/generate_jwt_keys.py data/keys
    else
        echo "WARNING: JWT key generator not found."
        echo "You may need to generate JWT keys manually."
    fi
fi

# Set up environment file
echo "Setting up environment configuration..."
cat > data/config/.env << EOF
# ArcVault Docker Configuration
CONTROLLER_PORT=8501
CONTROLLER_ADDRESS=0.0.0.0
STREAMLIT_HEADLESS=1
AGENT_VERIFY_SSL=false
AGENT_REQUIRE_JWT=true
AGENT_JWT_PUBLIC_KEY_PATH=/app/keys/controller_public.pem
AGENT_JWT_ISSUER=backup-controller
AGENT_JWT_AUDIENCE=backup-agent
CTRL_JWT_PRIVATE_KEY_PATH=/app/keys/controller_private.pem
CTRL_JWT_ISSUER=backup-controller
CTRL_JWT_AUDIENCE=backup-agent
EOF

# Build and start services
echo "Building and starting ArcVault server..."
docker-compose up -d --build

if [ $? -ne 0 ]; then
    echo "ERROR: Failed to build or start Docker services."
    echo "Check Docker is running and try again."
    exit 1
fi

# Wait for service to be ready
echo "Waiting for service to be ready..."
sleep 10

# Check if service is running
if docker-compose ps | grep -q "arcvault-server.*Up"; then
    echo
    echo "========================================"
    echo "ArcVault Server Deployment Complete!"
    echo "========================================"
    echo
    echo "✅ ArcVault server is running!"
    echo
    echo "Access points:"
    echo "- Web Interface: http://localhost:8501"
    echo "- With Nginx (HTTPS): https://localhost:443 (if enabled)"
    echo
    echo "Management commands:"
    echo "- View logs: docker-compose logs -f"
    echo "- Stop service: docker-compose down"
    echo "- Restart service: docker-compose restart"
    echo "- Update service: docker-compose up -d --build"
    echo
    echo "Configuration files:"
    echo "- Environment: data/config/.env"
    echo "- JWT Keys: data/keys/"
    echo "- Logs: data/logs/"
    echo
    echo "To enable HTTPS with Nginx:"
    echo "docker-compose --profile with-nginx up -d"
    echo
    echo "Client Deployment:"
    echo "- Use ArcVault-Client-Installer.exe on Windows machines"
    echo "- Configure agents to connect to this server"
    echo "- Add agents through the web interface"
else
    echo "❌ Failed to start ArcVault server."
    echo "Check logs with: docker-compose logs"
    exit 1
fi