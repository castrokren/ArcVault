# ArcVault Configuration Files

This directory contains default configuration templates used during deployment.

## Configuration Files

### `default_service_config.json`
Service configuration for the ArcVault Agent. This file controls:
- Network settings (host, port)
- JWT authentication
- SSL/TLS configuration
- Logging levels

**Deployed to**: `C:\ProgramData\ArcVault\service_config.json`

### `default_backup_config.json`
Default backup job configuration template. Contains:
- Example backup job definitions
- Global settings
- Retention policies
- Notification settings

**Deployed to**: `C:\ProgramData\ArcVault\backup_config.json`

### `service.env`
Environment variables for the Windows service. Includes:
- Network configuration
- Authentication settings
- SSL/TLS paths
- Performance tuning

**Deployed to**: `C:\ProgramData\ArcVault\.env`

## Customization

You can customize these files before deployment:

1. Edit the configuration files in this directory
2. Run the deployment script: `.\Deploy-ArcVault.ps1`
3. Your custom configurations will be deployed

## Post-Deployment Configuration

After deployment, you can modify configurations at:
- Service config: `C:\ProgramData\ArcVault\service_config.json`
- Backup config: `C:\ProgramData\ArcVault\backup_config.json`
- Environment: `C:\ProgramData\ArcVault\.env`

**Note**: After changing configurations, restart the service:
```powershell
Restart-Service ArcVaultAgent
```

## Configuration Priority

The agent checks configuration in this order:
1. **service_config.json** - Highest priority
2. **Environment variables** - From `.env` file
3. **Hardcoded defaults** - Fallback values

## Security Notes

- Keep JWT private keys secure
- Use strong SSL certificates in production
- Restrict file permissions to Administrators
- Change default ports if needed for security

