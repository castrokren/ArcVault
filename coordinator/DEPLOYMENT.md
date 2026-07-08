# Production Deployment Guide

## Prerequisites

1. TLS certificates (generate with `coordinator init`)
2. Database directory (writable)
3. Environment variables for sensitive credentials

## Environment Variables

Before starting the coordinator in any environment, set:

```bash
export ARCVAULT_ADMIN_TOKEN=$(openssl rand -hex 32)
export ARCVAULT_JWT_SECRET=$(openssl rand -hex 32)
```

### Security Best Practices

1. **Never commit tokens to git**
   - Add `.env` and `config.json` to `.gitignore`

2. **Use a secrets manager in production**
   - HashiCorp Vault
   - AWS Secrets Manager
   - Azure Key Vault
   - Google Cloud Secret Manager

3. **Rotate tokens regularly**
   - Store previous token in Vault for graceful rotation
   - Plan 24-hour migration window for token change

4. **Audit token access**
   - Log when tokens are loaded from environment
   - Log when CORS origins are validated

## Startup Procedure

### Development

```bash
# Generate tokens (one-time)
export ARCVAULT_ADMIN_TOKEN=$(openssl rand -hex 32)
export ARCVAULT_JWT_SECRET=$(openssl rand -hex 32)

# Start server
coordinator start
```

### Production (Linux/macOS with systemd)

Create `/etc/systemd/system/arcvault.service`:

```ini
[Unit]
Description=ArcVault Coordinator
After=network.target

[Service]
Type=simple
User=arcvault
WorkingDirectory=/opt/arcvault
ExecStart=/opt/arcvault/coordinator start
Restart=on-failure
RestartSec=10

# Load environment variables from secure location
EnvironmentFile=/etc/arcvault/.env

[Install]
WantedBy=multi-user.target
```

Create `/etc/arcvault/.env`:

```bash
ARCVAULT_ADMIN_TOKEN=<token-from-vault>
ARCVAULT_JWT_SECRET=<secret-from-vault>
```

Set restrictive permissions:

```bash
chmod 600 /etc/arcvault/.env
chown arcvault:arcvault /etc/arcvault/.env
```

Start service:

```bash
systemctl enable arcvault
systemctl start arcvault
```

### Production (Docker)

Pass environment variables at runtime:

```bash
docker run -d \
  -p 443:8443 \
  -v /data/arcvault:/root/.arcvault \
  -e ARCVAULT_ADMIN_TOKEN=$(vault read -field=token secret/arcvault/admin) \
  -e ARCVAULT_JWT_SECRET=$(vault read -field=secret secret/arcvault/jwt) \
  arcvault/coordinator:latest
```

## Startup Validation

After starting, verify in logs:

```bash
# Should see:
# [config] AdminToken loaded from ARCVAULT_ADMIN_TOKEN env var
# [config] JWTSecret loaded from ARCVAULT_JWT_SECRET env var
```

If you see warnings about environment variables not set:

```
CRITICAL: AdminToken not set. Set ARCVAULT_ADMIN_TOKEN environment variable
```

Then:
1. Stop the coordinator
2. Export the environment variables
3. Restart

## Token Rotation Procedure

To change admin token without downtime:

1. Generate new token:
   ```bash
   NEW_TOKEN=$(openssl rand -hex 32)
   echo "New token: $NEW_TOKEN"
   ```

2. Store in secret manager (don't lose it!)

3. Update environment:
   ```bash
   export ARCVAULT_ADMIN_TOKEN=$NEW_TOKEN
   systemctl restart arcvault
   ```

4. Update all agent configurations to use new token

5. After 24 hours, old token can be discarded

## Troubleshooting

### "config.json contains tokens" warning

If you see this, the config file was created by an older version:

```bash
# Regenerate with sanitized config:
rm config.json
coordinator init

# Set env vars
export ARCVAULT_ADMIN_TOKEN=<new-token>
export ARCVAULT_JWT_SECRET=<new-secret>

# Start
coordinator start
```

### Connection refused on startup

Check environment variables are exported:

```bash
echo $ARCVAULT_ADMIN_TOKEN
echo $ARCVAULT_JWT_SECRET
```

If empty, export them and restart.

## CORS Configuration

The coordinator validates WebSocket and API origins against `AllowedOrigins` in the config file.

### Development

For development, run `coordinator init` and the system will use default localhost origins:
- `http://localhost:5173` (Vite dev server)
- `http://localhost:3000` (alternative dashboard port)

### Production

Edit `config.json` and set `allowed_origins`:

```json
{
  "allowed_origins": [
    "https://dashboard.example.com",
    "https://dashboard-backup.example.com"
  ]
}
```

**Security notes:**
- Wildcard `"*"` is NOT allowed
- Only HTTPS origins are allowed in production (except localhost)
- Empty list in production will cause startup failure

## WebSocket Validation

The coordinator validates WebSocket connection origins:

1. Browser connects from `https://dashboard.example.com`
2. WebSocket upgrade includes `Origin: https://dashboard.example.com` header
3. Coordinator checks if origin is in `AllowedOrigins` list
4. If allowed, connection succeeds
5. If denied, connection fails with HTTP 403

Watch logs for origin validation:
```
[WebSocket] Accepted origin: "https://dashboard.example.com"
[WebSocket] Rejected origin: "https://attacker.com" (not in AllowedOrigins)
```

## Monitoring

### Key Log Messages

```
[config] AdminToken loaded from ARCVAULT_ADMIN_TOKEN env var
[config] JWTSecret loaded from ARCVAULT_JWT_SECRET env var
[config] Development: Using default AllowedOrigins
[WebSocket] Accepted origin: "https://dashboard.example.com"
[CORS] Rejected origin: "https://attacker.com" (not in whitelist)
```

### Health Check Endpoint

```bash
curl -k https://coordinator:443/api/health
```

Should return 200 with status information.

## Post-Deployment Verification

1. **Environment Variables Loaded**
   ```
   ✓ See "[config] AdminToken loaded from ARCVAULT_ADMIN_TOKEN env var" in logs
   ```

2. **CORS Origins Validated**
   ```
   ✓ Dashboard connects successfully
   ✓ See "[WebSocket] Accepted origin" in logs
   ```

3. **WebSocket Connection**
   ```
   ✓ Dashboard shows "Connected" status
   ✓ Real-time updates received
   ```

4. **API Access**
   ```bash
   curl -k -H "Authorization: Bearer $ARCVAULT_ADMIN_TOKEN" \
     https://coordinator:443/api/jobs
   # Should return job list
   ```
