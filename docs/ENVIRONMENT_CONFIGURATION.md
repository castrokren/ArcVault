# ArcVault Environment Configuration Guide

This guide explains how to properly configure environment variables for ArcVault in different deployment scenarios.

## Quick Start

### 1. Initialize Environment
```bash
# For development
python scripts/setup_environment.py init development

# For production
python scripts/setup_environment.py init production
python scripts/setup_environment.py keys
```

### 2. Validate Configuration
```bash
python scripts/setup_environment.py validate
```

## Environment Types

### Development Environment
- **Purpose**: Local development and testing
- **Authentication**: Simple token-based (insecure)
- **SSL**: Optional
- **Logging**: Verbose debug output

### Staging Environment
- **Purpose**: Pre-production testing
- **Authentication**: JWT with RSA keys (recommended)
- **SSL**: Optional but recommended
- **Logging**: Debug level for troubleshooting

### Production Environment
- **Purpose**: Live production deployment
- **Authentication**: JWT with RSA keys (required)
- **SSL**: Required for security
- **Logging**: Info level for performance

## Environment Variables Reference

### Agent Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `AGENT_LISTEN_HOST` | Host to bind to | `0.0.0.0` | No |
| `AGENT_PORT` | Port to listen on | `8443` | No |
| `AGENT_PUBLIC_HOST` | Public hostname for discovery | `localhost` | No |
| `AGENT_PUBLIC_URL` | Full public URL | `http://localhost:8443` | No |
| `AGENT_PUBLIC_SCHEME` | URL scheme (http/https) | `http` | No |

### Authentication

#### Simple Token (Development Only)
| Variable | Description | Example |
|----------|-------------|---------|
| `AGENT_TOKEN` | Static authentication token | `dev-token-12345` |
| `AGENT_REQUIRE_JWT` | Force JWT authentication | `false` |

#### JWT Authentication (Production)
| Variable | Description | Example |
|----------|-------------|---------|
| `AGENT_REQUIRE_JWT` | Force JWT authentication | `true` |
| `AGENT_JWT_PUBLIC_KEY_PATH` | Path to controller's public key | `keys/controller_public.pem` |
| `AGENT_JWT_ISSUER` | JWT issuer to validate | `backup-controller` |
| `AGENT_JWT_AUDIENCE` | JWT audience to validate | `backup-agent` |
| `AGENT_JWT_LEEWAY` | Clock skew tolerance (seconds) | `60` |

### SSL/TLS Configuration

| Variable | Description | Example |
|----------|-------------|---------|
| `AGENT_CERTFILE` | SSL certificate file | `keys/agent.crt` |
| `AGENT_KEYFILE` | SSL private key file | `keys/agent.key` |
| `AGENT_VERIFY_SSL` | Verify SSL certificates | `true` |
| `AGENT_CA_BUNDLE` | CA certificate bundle | `keys/ca-bundle.crt` |

### Controller Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `CONTROLLER_PORT` | Streamlit port | `8501` | No |
| `CONTROLLER_ADDRESS` | Host to bind to | `0.0.0.0` | No |
| `STREAMLIT_HEADLESS` | Run without browser | `0` | No |
| `AGENT_URL` | Default agent URL | `http://localhost:8443` | No |

### Backup Engine Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `ROBOCOPY_DEFAULT_RETRIES` | Default retry count | `1` |
| `ROBOCOPY_DEFAULT_WAIT` | Wait time between retries | `5` |
| `ROBOCOPY_DEFAULT_THREADS` | Number of threads | `12` |
| `ROBOCOPY_DEFAULT_VERBOSE` | Verbose output | `true` |
| `ROBOCOPY_DEFAULT_INCLUDE_EMPTY` | Include empty directories | `true` |
| `ROBOCOPY_DEFAULT_REPORT_EXTRAS` | Report extra files | `true` |
| `ROBOCOPY_DEFAULT_REMOVE_SYSTEM_HIDDEN` | Remove system/hidden files | `true` |

### Logging Configuration

| Variable | Description | Options |
|----------|-------------|---------|
| `BACKUP_LOG_LEVEL` | Log level for backup operations | `DEBUG`, `INFO`, `WARNING`, `ERROR` |
| `BACKUP_LOG_DIR` | Directory for backup logs | `#backup_logs` |
| `LOG_LEVEL` | General application log level | `DEBUG`, `INFO`, `WARNING`, `ERROR` |
| `DEBUG` | Enable debug mode | `true`, `false` |

## Security Best Practices

### 1. JWT Key Management
- Generate unique key pairs for each environment
- Store private keys securely (600 permissions)
- Rotate keys regularly
- Never commit keys to version control

### 2. SSL/TLS Configuration
- Use valid certificates from trusted CAs
- Enable certificate verification in production
- Use strong cipher suites
- Keep certificates up to date

### 3. Network Security
- Use firewalls to restrict access
- Bind to specific interfaces when possible
- Use VPNs for remote access
- Monitor network traffic

### 4. Token Security
- Use long, random tokens
- Rotate tokens regularly
- Never use default tokens in production
- Store tokens securely

## Deployment Examples

### Development Setup
```bash
# 1. Initialize development environment
python scripts/setup_environment.py init development

# 2. Start agent
cd apps/backup-agent
python -m venv .venv
.venv\Scripts\activate
pip install -e .
uvicorn agent.app:app --host 0.0.0.0 --port 8443

# 3. Start controller (in another terminal)
cd apps/backup-control
python -m venv .venv
.venv\Scripts\activate
pip install -e .
streamlit run controller/app.py
```

### Production Setup
```bash
# 1. Initialize production environment
python scripts/setup_environment.py init production

# 2. Generate JWT keys
python scripts/setup_environment.py keys

# 3. Set up SSL certificates
# (Copy your certificates to keys/ directory)

# 4. Update .env with production values
# Edit .env file with your hostnames and paths

# 5. Validate configuration
python scripts/setup_environment.py validate

# 6. Build and deploy
cd apps/backup-agent
scripts/build_agent_exe.bat
cd ../backup-control
scripts/build_controller_exe.bat
```

## Troubleshooting

### Common Issues

#### 1. "JWT not configured" Error
- Ensure `AGENT_JWT_PUBLIC_KEY_PATH` points to a valid file
- Check that the public key file exists and is readable
- Verify the key format is correct (PEM)

#### 2. "Invalid JWT" Error
- Check that `CTRL_JWT_PRIVATE_KEY_PATH` is set correctly
- Ensure the private key matches the public key
- Verify JWT issuer and audience settings match

#### 3. SSL Certificate Errors
- Ensure certificate files exist and are readable
- Check certificate validity and expiration
- Verify the certificate matches the hostname
- Ensure CA bundle is correct

#### 4. Connection Refused
- Check that the agent is running on the correct port
- Verify firewall settings
- Ensure the URL in `AGENT_URL` is correct
- Check network connectivity

### Validation Commands
```bash
# Check configuration
python scripts/setup_environment.py validate

# Test agent connectivity
curl http://localhost:8443/health

# Test controller
curl http://localhost:8501
```

## Environment File Examples

### Minimal Development (.env)
```bash
AGENT_TOKEN=dev-token-12345
AGENT_REQUIRE_JWT=false
AGENT_VERIFY_SSL=false
```

### Production (.env)
```bash
AGENT_REQUIRE_JWT=true
AGENT_JWT_PUBLIC_KEY_PATH=/opt/arcvault/keys/controller_public.pem
AGENT_JWT_ISSUER=backup-controller
AGENT_JWT_AUDIENCE=backup-agent
AGENT_CERTFILE=/opt/arcvault/keys/agent.crt
AGENT_KEYFILE=/opt/arcvault/keys/agent.key
AGENT_VERIFY_SSL=true
CTRL_JWT_PRIVATE_KEY_PATH=/opt/arcvault/keys/controller_private.pem
```

## Additional Resources

- [JWT Authentication Guide](JWT_AUTHENTICATION.md)
- [SSL/TLS Setup Guide](SSL_SETUP.md)
- [Deployment Guide](DEPLOYMENT.md)
- [Troubleshooting Guide](TROUBLESHOOTING.md)
