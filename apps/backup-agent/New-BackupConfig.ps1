<#
.SYNOPSIS
    Interactive configuration generator for ArcVault Backup Agent.

.DESCRIPTION
    This script helps you create backup configurations by asking simple questions
    and generating a properly formatted JSON configuration file.

.PARAMETER Template
    Template to use as a base (optional). If not specified, shows a menu.

.PARAMETER OutputFile
    Name of the output configuration file (default: backup_config.json)

.PARAMETER Interactive
    Run in interactive mode (default: true)

.EXAMPLE
    .\New-BackupConfig.ps1
    # Interactive mode - asks questions and generates config

.EXAMPLE
    .\New-BackupConfig.ps1 -Template simple -OutputFile my_backup.json
    # Uses simple template, saves to my_backup.json

.EXAMPLE
    .\New-BackupConfig.ps1 -Template network -OutputFile nas_backup.json
    # Uses network template for NAS backup

.NOTES
    Version: 1.0
    Author: ArcVault Team
    Last Updated: 2025-12-02
#>

param(
    [Parameter(Mandatory=$false)]
    [ValidateSet('simple', 'multiple', 'network', 'performance', 'reliable', 'minimal')]
    [string]$Template,

    [Parameter(Mandatory=$false)]
    [string]$OutputFile = "backup_config.json",

    [Parameter(Mandatory=$false)]
    [bool]$Interactive = $true
)

# Color functions
function Write-Header {
    param([string]$Message)
    Write-Host "`n========================================" -ForegroundColor Cyan
    Write-Host " $Message" -ForegroundColor Cyan
    Write-Host "========================================`n" -ForegroundColor Cyan
}

function Write-Step {
    param([string]$Message)
    Write-Host "▶ $Message" -ForegroundColor Yellow
}

function Write-Success {
    param([string]$Message)
    Write-Host "✓ $Message" -ForegroundColor Green
}

function Write-Error {
    param([string]$Message)
    Write-Host "✗ $Message" -ForegroundColor Red
}

function Write-Info {
    param([string]$Message)
    Write-Host "ℹ $Message" -ForegroundColor Blue
}

# Template configurations
$templates = @{
    'simple' = @{
        Name = 'Simple Local Backup'
        File = 'simple_local_backup.json'
        Description = 'Basic backup from one folder to a local drive'
        UseCases = @('Personal documents', 'Single folder backups', 'Local disk-to-disk')
    }
    'multiple' = @{
        Name = 'Multiple Sources Backup'
        File = 'multiple_sources_backup.json'
        Description = 'Back up multiple folders to one destination'
        UseCases = @('System backups', 'Multiple project folders', 'User profile backups')
    }
    'network' = @{
        Name = 'Network Share Backup'
        File = 'network_share_backup.json'
        Description = 'Backup to network location (NAS, file server)'
        UseCases = @('NAS backups', 'File server backups', 'Shared storage')
    }
    'performance' = @{
        Name = 'High Performance Backup'
        File = 'high_performance_backup.json'
        Description = 'Fast backups on high-speed storage'
        UseCases = @('Large files', 'SSD to SSD', 'Time-critical backups')
    }
    'reliable' = @{
        Name = 'Conservative/Reliable Backup'
        File = 'conservative_reliable_backup.json'
        Description = 'Backups over slow or unreliable networks'
        UseCases = @('VPN connections', 'Slow networks', 'Remote offices')
    }
    'minimal' = @{
        Name = 'Minimal Backup'
        File = 'minimal_backup.json'
        Description = 'Simplest possible configuration'
        UseCases = @('Quick testing', 'Learning', 'Simple needs')
    }
}

# Display welcome banner
Write-Header "ArcVault Backup Configuration Generator"
Write-Host "This tool will help you create a backup configuration file.`n"

# Select template
if (-not $Template) {
    Write-Step "Select a template:"
    Write-Host ""
    
    $i = 1
    $templateKeys = @()
    foreach ($key in $templates.Keys | Sort-Object) {
        $t = $templates[$key]
        Write-Host "  $i. " -NoNewline -ForegroundColor Cyan
        Write-Host "$($t.Name)" -ForegroundColor White
        Write-Host "     $($t.Description)" -ForegroundColor Gray
        Write-Host "     Best for: " -NoNewline -ForegroundColor DarkGray
        Write-Host ($t.UseCases -join ", ") -ForegroundColor Gray
        Write-Host ""
        $templateKeys += $key
        $i++
    }
    
    do {
        $selection = Read-Host "Enter number (1-$($templates.Count))"
        $selectedIndex = [int]$selection - 1
    } while ($selectedIndex -lt 0 -or $selectedIndex -ge $templates.Count)
    
    $Template = $templateKeys[$selectedIndex]
}

$selectedTemplate = $templates[$Template]
Write-Success "Selected template: $($selectedTemplate.Name)"

# Load template
$templatePath = Join-Path "example_configs" $selectedTemplate.File
if (-not (Test-Path $templatePath)) {
    Write-Error "Template file not found: $templatePath"
    Write-Info "Make sure you're running this script from the backup-agent directory."
    exit 1
}

try {
    $config = Get-Content $templatePath -Raw | ConvertFrom-Json
    Write-Success "Template loaded successfully"
} catch {
    Write-Error "Failed to load template: $_"
    exit 1
}

# Interactive configuration
if ($Interactive) {
    Write-Header "Configuration Setup"
    
    # Sources
    Write-Step "Configure source directories (what to back up)"
    Write-Info "Current sources: $($config.sources -join ', ')"
    $changeSources = Read-Host "Change sources? (y/n)"
    
    if ($changeSources -eq 'y') {
        $sources = @()
        $addingMore = $true
        
        while ($addingMore) {
            $source = Read-Host "Enter source path (or press Enter to finish)"
            if ([string]::IsNullOrWhiteSpace($source)) {
                $addingMore = $false
            } else {
                if (Test-Path $source) {
                    $sources += $source
                    Write-Success "Added: $source"
                } else {
                    Write-Error "Path does not exist: $source"
                    $retry = Read-Host "Add anyway? (y/n)"
                    if ($retry -eq 'y') {
                        $sources += $source
                        Write-Success "Added: $source (unverified)"
                    }
                }
            }
        }
        
        if ($sources.Count -gt 0) {
            $config.sources = $sources
            Write-Success "Sources updated"
        }
    }
    
    # Destination
    Write-Step "Configure destination directory (where to back up)"
    Write-Info "Current destination: $($config.destination)"
    $changeDest = Read-Host "Change destination? (y/n)"
    
    if ($changeDest -eq 'y') {
        $destination = Read-Host "Enter destination path"
        
        if (Test-Path $destination) {
            # Test write access
            $testFile = Join-Path $destination ".write_test_$(Get-Random)"
            try {
                New-Item -ItemType File -Path $testFile -Force | Out-Null
                Remove-Item $testFile -Force
                $config.destination = $destination
                Write-Success "Destination updated and verified writable"
            } catch {
                Write-Error "Destination exists but is not writable: $_"
                $useAnyway = Read-Host "Use anyway? (y/n)"
                if ($useAnyway -eq 'y') {
                    $config.destination = $destination
                    Write-Success "Destination updated (unverified)"
                }
            }
        } else {
            Write-Error "Destination does not exist: $destination"
            $create = Read-Host "Create it? (y/n)"
            if ($create -eq 'y') {
                try {
                    New-Item -ItemType Directory -Path $destination -Force | Out-Null
                    $config.destination = $destination
                    Write-Success "Destination created and updated"
                } catch {
                    Write-Error "Failed to create destination: $_"
                }
            } else {
                $useAnyway = Read-Host "Use anyway? (y/n)"
                if ($useAnyway -eq 'y') {
                    $config.destination = $destination
                    Write-Success "Destination updated (unverified)"
                }
            }
        }
    }
    
    # Options
    Write-Step "Configure backup options"
    $changeOptions = Read-Host "Adjust performance settings? (y/n)"
    
    if ($changeOptions -eq 'y') {
        Write-Host "`nPerformance Settings:" -ForegroundColor Cyan
        
        # Threads
        Write-Info "Current threads: $($config.options.threads)"
        Write-Host "  Suggested: 4-8 for network, 12-16 for local, 64-128 for SSD" -ForegroundColor DarkGray
        $threads = Read-Host "Number of threads (1-128, or Enter to keep current)"
        if (-not [string]::IsNullOrWhiteSpace($threads)) {
            $config.options.threads = [int]$threads
        }
        
        # Retries
        Write-Info "Current retries: $($config.options.retries)"
        Write-Host "  Suggested: 1-3 for local, 5-10 for network, 20-50 for unreliable" -ForegroundColor DarkGray
        $retries = Read-Host "Number of retries (0-1000, or Enter to keep current)"
        if (-not [string]::IsNullOrWhiteSpace($retries)) {
            $config.options.retries = [int]$retries
        }
        
        # Wait seconds
        Write-Info "Current wait seconds: $($config.options.wait_seconds)"
        Write-Host "  Suggested: 3-5 for local, 10-15 for network, 20-30 for unreliable" -ForegroundColor DarkGray
        $wait = Read-Host "Wait seconds between retries (0-60, or Enter to keep current)"
        if (-not [string]::IsNullOrWhiteSpace($wait)) {
            $config.options.wait_seconds = [int]$wait
        }
    }
    
    # Credentials (for network backups)
    if ($config.destination -like "\\*") {
        Write-Step "Configure network credentials"
        Write-Info "Destination appears to be a network path"
        $needsCreds = Read-Host "Add credentials for destination? (y/n)"
        
        if ($needsCreds -eq 'y') {
            $username = Read-Host "Username"
            $password = Read-Host "Password" -AsSecureString
            $passwordPlain = [System.Runtime.InteropServices.Marshal]::PtrToStringAuto(
                [System.Runtime.InteropServices.Marshal]::SecureStringToBSTR($password)
            )
            $domain = Read-Host "Domain (or press Enter for WORKGROUP)"
            if ([string]::IsNullOrWhiteSpace($domain)) {
                $domain = "WORKGROUP"
            }
            
            $config | Add-Member -NotePropertyName "destination_credentials" -NotePropertyValue @{
                username = $username
                password = $passwordPlain
                domain = $domain
            } -Force
            
            Write-Success "Credentials added (will be encrypted when loaded)"
        }
    }
}

# Add metadata
$config | Add-Member -NotePropertyName "_generated_by" -NotePropertyValue "New-BackupConfig.ps1" -Force
$config | Add-Member -NotePropertyName "_generated_at" -NotePropertyValue (Get-Date -Format "yyyy-MM-dd HH:mm:ss") -Force
$config | Add-Member -NotePropertyName "_template" -NotePropertyValue $Template -Force

# Save configuration
Write-Header "Saving Configuration"

try {
    $config | ConvertTo-Json -Depth 10 | Set-Content -Path $OutputFile -Encoding UTF8
    Write-Success "Configuration saved to: $OutputFile"
} catch {
    Write-Error "Failed to save configuration: $_"
    exit 1
}

# Summary
Write-Header "Configuration Summary"
Write-Host "Template:    $($selectedTemplate.Name)" -ForegroundColor White
Write-Host "Sources:     $($config.sources.Count) directory(ies)" -ForegroundColor White
foreach ($src in $config.sources) {
    Write-Host "             - $src" -ForegroundColor Gray
}
Write-Host "Destination: $($config.destination)" -ForegroundColor White
Write-Host "Threads:     $($config.options.threads)" -ForegroundColor White
Write-Host "Retries:     $($config.options.retries)" -ForegroundColor White
Write-Host "Credentials: " -NoNewline -ForegroundColor White
if ($config.destination_credentials) {
    Write-Host "Yes (for destination)" -ForegroundColor Green
} else {
    Write-Host "No" -ForegroundColor Gray
}
Write-Host "`nOutput file: $OutputFile" -ForegroundColor Cyan

# Next steps
Write-Header "Next Steps"
Write-Host "1. Review the configuration file:"
Write-Host "   notepad $OutputFile" -ForegroundColor Yellow
Write-Host ""
Write-Host "2. Test the configuration:"
Write-Host "   python -m agent.engine.config_loader test $OutputFile" -ForegroundColor Yellow
Write-Host ""
Write-Host "3. Run your backup:"
Write-Host "   Use the ArcVault API or CLI to submit the backup job" -ForegroundColor Yellow
Write-Host ""
Write-Host "4. For more help:"
Write-Host "   See QUICK_START_CONFIGS.md" -ForegroundColor Yellow
Write-Host ""

Write-Success "Configuration generation complete!"

