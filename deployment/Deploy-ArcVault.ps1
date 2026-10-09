<#
.SYNOPSIS
    Professional deployment script for ArcVault Backup System

.DESCRIPTION
    Comprehensive deployment script that:
    - Performs pre-flight checks
    - Backs up existing installation
    - Installs Python environment and dependencies
    - Deploys application files
    - Configures Windows service
    - Generates SSL certificates and JWT keys
    - Verifies installation
    - Provides rollback capability

.PARAMETER InstallPath
    Installation directory for ArcVault (default: C:\Program Files\ArcVault)

.PARAMETER DataPath
    Data directory for configurations and keys (default: C:\ProgramData\ArcVault)

.PARAMETER Port
    Agent service port (default: 8443)

.PARAMETER SourcePath
    Source directory containing ArcVault application files (default: ../apps/backup-agent)

.PARAMETER Development
    Install in development mode (no service, console mode)

.PARAMETER Silent
    Silent installation with minimal output

.PARAMETER ConfigureOnly
    Only run configuration, don't install files

.PARAMETER SkipBackup
    Skip backup of existing installation

.PARAMETER AutoInstallPython
    Automatically download and install Python 3.11 if not found (no prompt)

.PARAMETER LogPath
    Path to installation log file

.EXAMPLE
    .\Deploy-ArcVault.ps1
    Standard installation to default location

.EXAMPLE
    .\Deploy-ArcVault.ps1 -AutoInstallPython
    Automatically install Python if needed (no prompts)

.EXAMPLE
    .\Deploy-ArcVault.ps1 -InstallPath "D:\ArcVault" -Port 8080
    Custom installation path and port

.EXAMPLE
    .\Deploy-ArcVault.ps1 -Development
    Development mode installation

.EXAMPLE
    .\Deploy-ArcVault.ps1 -Silent -LogPath "C:\Logs\install.log"
    Silent installation with custom log

.NOTES
    Author: ArcVault Team
    Version: 1.0.0
    Requires: PowerShell 5.1+, Administrator privileges
#>

[CmdletBinding()]
param(
    [string]$InstallPath = "C:\Program Files\ArcVault",
    [string]$DataPath = "C:\ProgramData\ArcVault",
    [int]$Port = 8443,
    [string]$SourcePath = "..\apps\backup-agent",
    [switch]$Development,
    [switch]$Silent,
    [switch]$ConfigureOnly,
    [switch]$SkipBackup,
    [switch]$AutoInstallPython,
    [string]$LogPath = "$env:TEMP\arcvault_deployment.log"
)

# Script configuration
$ErrorActionPreference = "Stop"
$ProgressPreference = if ($Silent) { "SilentlyContinue" } else { "Continue" }
$script:StartTime = Get-Date
$script:InstallationSuccess = $false

# Version information
$script:Version = "1.0.0"
$script:RequiredPSVersion = "5.1"
$script:RequiredPythonVersion = "3.9"

#region Logging Functions

function Write-Log {
    param(
        [string]$Message,
        [ValidateSet("Info", "Success", "Warning", "Error")]
        [string]$Level = "Info"
    )
    
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    $logMessage = "[$timestamp] [$Level] $Message"
    
    # Write to log file
    Add-Content -Path $LogPath -Value $logMessage -ErrorAction SilentlyContinue
    
    # Write to console if not silent
    if (-not $Silent) {
        switch ($Level) {
            "Info"    { Write-Host $Message -ForegroundColor Cyan }
            "Success" { Write-Host "[OK] $Message" -ForegroundColor Green }
            "Warning" { Write-Warning $Message }
            "Error"   { Write-Error $Message }
        }
    }
}

function Write-Step {
    param([string]$Message)
    Write-Log "`n=== $Message ===" -Level Info
}

function Write-Success {
    param([string]$Message)
    Write-Log $Message -Level Success
}

function Write-InstallError {
    param([string]$Message, [System.Management.Automation.ErrorRecord]$ErrorRecord)
    Write-Log $Message -Level Error
    if ($ErrorRecord) {
        Write-Log "Error Details: $($ErrorRecord.Exception.Message)" -Level Error
        Write-Log "Stack Trace: $($ErrorRecord.ScriptStackTrace)" -Level Error
    }
}

#endregion

#region Pre-flight Checks

function Test-Administrator {
    Write-Log "Checking administrator privileges..."
    $currentPrincipal = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
    $isAdmin = $currentPrincipal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
    
    if (-not $isAdmin) {
        Write-InstallError "This script requires administrator privileges. Please run as Administrator."
        exit 1
    }
    Write-Success "Administrator privileges confirmed"
}

function Test-PowerShellVersion {
    Write-Log "Checking PowerShell version..."
    $version = $PSVersionTable.PSVersion
    Write-Log "PowerShell version: $($version.ToString())"
    
    if ($version.Major -lt 5 -or ($version.Major -eq 5 -and $version.Minor -lt 1)) {
        Write-InstallError "PowerShell 5.1 or later required. Current version: $($version.ToString())"
        exit 1
    }
    Write-Success "PowerShell version check passed"
}

function Test-DiskSpace {
    param([string]$Path, [int]$RequiredGB = 2)
    
    Write-Log "Checking disk space for: $Path"
    $drive = (Get-Item $Path -ErrorAction SilentlyContinue).PSDrive.Name
    if (-not $drive) {
        $drive = $Path.Substring(0, 1)
    }
    
    $disk = Get-PSDrive $drive -ErrorAction SilentlyContinue
    if (-not $disk) {
        Write-Log "Unable to check disk space for $Path" -Level Warning
        return $true
    }
    
    $freeGB = [math]::Round($disk.Free / 1GB, 2)
    Write-Log "Available space on ${drive}: $freeGB GB"
    
    if ($freeGB -lt $RequiredGB) {
        Write-InstallError "Insufficient disk space. Required: ${RequiredGB}GB, Available: ${freeGB}GB"
        return $false
    }
    Write-Success "Disk space check passed: ${freeGB}GB available"
    return $true
}

function Test-PortAvailable {
    param([int]$TestPort)
    
    Write-Log "Checking if port $TestPort is available..."
    $listener = $null
    try {
        $listener = New-Object System.Net.Sockets.TcpListener([System.Net.IPAddress]::Any, $TestPort)
        $listener.Start()
        $listener.Stop()
        Write-Success "Port $TestPort is available"
        return $true
    }
    catch {
        Write-Log "Port $TestPort is already in use" -Level Warning
        return $false
    }
    finally {
        if ($listener) { $listener.Stop() }
    }
}

function Test-SourceFiles {
    Write-Log "Checking source files..."
    
    # Resolve source path relative to script location
    $scriptDir = Split-Path -Parent $PSCommandPath
    $resolvedSourcePath = Join-Path $scriptDir $SourcePath
    $script:SourcePath = $resolvedSourcePath
    
    if (-not (Test-Path $resolvedSourcePath)) {
        Write-InstallError "Source path not found: $resolvedSourcePath"
        Write-Log "Looking for backup-agent directory..." -Level Info
        
        # Try to find backup-agent directory
        $possiblePaths = @(
            "..\apps\backup-agent",
            "..\..\apps\backup-agent",
            "..\backup-agent",
            "backup-agent"
        )
        
        foreach ($path in $possiblePaths) {
            $testPath = Join-Path $scriptDir $path
            if (Test-Path $testPath) {
                Write-Log "Found source at: $testPath" -Level Info
                $script:SourcePath = $testPath
                return $true
            }
        }
        
        Write-InstallError "Cannot find ArcVault source files. Please specify -SourcePath"
        return $false
    }
    
    # Check for required files
    $requiredFiles = @(
        "arcvault_unified.py",
        "requirements_unified.txt",
        "agent",
        "service"
    )
    
    foreach ($file in $requiredFiles) {
        $filePath = Join-Path $resolvedSourcePath $file
        if (-not (Test-Path $filePath)) {
            Write-InstallError "Required file/folder missing: $file"
            return $false
        }
    }
    
    Write-Success "Source files validated"
    return $true
}

#endregion

#region Python Environment

function Install-Python {
    <#
    .SYNOPSIS
        Check for Python and install if needed with multiple fallback download methods
    #>
    
    Write-Log "Checking Python installation..."
    
    # Refresh PATH from registry to detect recently installed Python
    Write-Log "Refreshing environment variables..."
    $env:Path = [System.Environment]::GetEnvironmentVariable("Path", "Machine") + ";" + 
                [System.Environment]::GetEnvironmentVariable("Path", "User")
    
    # Check multiple Python commands (python, python3, py)
    $pythonCommands = @("python", "python3", "py")
    
    foreach ($cmd in $pythonCommands) {
        try {
            $pythonVersion = & $cmd --version 2>&1
            if ($pythonVersion -match "Python (\d+)\.(\d+)\.(\d+)") {
                $major = [int]$matches[1]
                $minor = [int]$matches[2]
                
                if ($major -ge 3 -and $minor -ge 9) {
                    Write-Log "Found: $pythonVersion (via '$cmd' command)"
                    Write-Success "Python $major.$minor meets requirements"
                    return $true
                }
                else {
                    Write-Log "Found $pythonVersion but version 3.9+ required" -Level Warning
                }
            }
        }
        catch {
            # Command not found, try next
        }
    }
    
    # Check common installation paths directly
    Write-Log "Checking common Python installation paths..."
    $commonPaths = @(
        "C:\Python311\python.exe",
        "C:\Python310\python.exe",
        "C:\Python39\python.exe",
        "$env:LOCALAPPDATA\Programs\Python\Python311\python.exe",
        "$env:LOCALAPPDATA\Programs\Python\Python310\python.exe",
        "$env:LOCALAPPDATA\Programs\Python\Python39\python.exe",
        "$env:ProgramFiles\Python311\python.exe",
        "$env:ProgramFiles\Python310\python.exe"
    )
    
    foreach ($path in $commonPaths) {
        if (Test-Path $path) {
            try {
                $pythonVersion = & $path --version 2>&1
                if ($pythonVersion -match "Python (\d+)\.(\d+)\.(\d+)") {
                    $major = [int]$matches[1]
                    $minor = [int]$matches[2]
                    
                    if ($major -ge 3 -and $minor -ge 9) {
                        Write-Log "Found: $pythonVersion at $path"
                        Write-Log "Python is installed but not in PATH. Adding to current session..." -Level Warning
                        
                        # Add to current session PATH
                        $pythonDir = Split-Path $path
                        $env:Path = "$pythonDir;$env:Path"
                        
                        Write-Success "Python $major.$minor detected and added to PATH"
                        return $true
                    }
                }
            }
            catch {
                # Skip if can't execute
            }
        }
    }
    
    Write-Log "Python not found in PATH or common installation locations" -Level Warning
    
    # Python not found or wrong version - offer automatic installation
    Write-Log "Python 3.9+ is required but not found" -Level Warning
    
    $autoInstall = $AutoInstallPython
    if (-not $Silent -and -not $AutoInstallPython) {
        $response = Read-Host "Would you like to automatically download and install Python 3.11? (Y/N)"
        $autoInstall = $response -eq "Y"
    }
    
    if (-not $autoInstall) {
        Write-Log "Manual installation required from: https://www.python.org/downloads/" -Level Info
        Write-InstallError "Python 3.9+ required. Please install and re-run this script."
        return $false
    }
    
    # Automatic Python installation
    Write-Log "`n=== Installing Python 3.11.9 ===" -Level Info
    
    $pythonVersion = "3.11.9"
    $pythonUrl = "https://www.python.org/ftp/python/$pythonVersion/python-$pythonVersion-amd64.exe"
    $installerPath = "$env:TEMP\python-$pythonVersion-installer.exe"
    
    # Force TLS 1.2 (required for python.org)
    Write-Log "Configuring secure connection..."
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
    
    # Try multiple download methods
    $downloaded = $false
    
    # Method 1: Invoke-WebRequest with TLS 1.2
    Write-Log "Downloading Python installer (Method 1: WebRequest)..." -Level Info
    try {
        $ProgressPreference = 'SilentlyContinue'  # Faster download
        Invoke-WebRequest -Uri $pythonUrl -OutFile $installerPath `
            -UseBasicParsing -TimeoutSec 300 -ErrorAction Stop
        
        if (Test-Path $installerPath) {
            $fileSize = (Get-Item $installerPath).Length / 1MB
            Write-Log "Downloaded successfully ($([math]::Round($fileSize, 2)) MB)" -Level Info
            $downloaded = $true
        }
    } catch {
        Write-Log "Method 1 failed: $($_.Exception.Message)" -Level Warning
    }
    
    # Method 2: BITS Transfer (Windows Background Intelligent Transfer)
    if (-not $downloaded) {
        Write-Log "Trying alternative download (Method 2: BITS)..." -Level Info
        try {
            Start-BitsTransfer -Source $pythonUrl -Destination $installerPath `
                -Description "Downloading Python $pythonVersion" -ErrorAction Stop
            
            if (Test-Path $installerPath) {
                Write-Success "Downloaded successfully via BITS"
                $downloaded = $true
            }
        } catch {
            Write-Log "Method 2 failed: $($_.Exception.Message)" -Level Warning
        }
    }
    
    # Method 3: .NET WebClient
    if (-not $downloaded) {
        Write-Log "Trying final download method (Method 3: WebClient)..." -Level Info
        try {
            $webClient = New-Object System.Net.WebClient
            $webClient.DownloadFile($pythonUrl, $installerPath)
            
            if (Test-Path $installerPath) {
                Write-Success "Downloaded successfully via WebClient"
                $downloaded = $true
            }
        } catch {
            Write-Log "Method 3 failed: $($_.Exception.Message)" -Level Warning
        }
    }
    
    # If all download methods failed
    if (-not $downloaded) {
        Write-Log "`nAll download methods failed!" -Level Warning
        Write-Log "`nManual installation options:" -Level Info
        Write-Log "  1. Download from: $pythonUrl" -Level Info
        Write-Log "  2. Or use: winget install Python.Python.3.11" -Level Info
        Write-Log "  3. Or use: choco install python311" -Level Info
        Write-Log "`nAfter manual installation, run this script again." -Level Info
        return $false
    }
    
    # Verify downloaded file
    Write-Log "Verifying installer..."
    if (-not (Test-Path $installerPath)) {
        Write-Log "Installer file not found at: $installerPath" -Level Warning
        return $false
    }
    
    $fileSize = (Get-Item $installerPath).Length
    if ($fileSize -lt 1MB) {
        Write-Log "Downloaded file is too small (corrupted?): $fileSize bytes" -Level Warning
        Remove-Item $installerPath -Force -ErrorAction SilentlyContinue
        return $false
    }
    
    Write-Success "Installer verified ($([math]::Round($fileSize/1MB, 2)) MB)"
    
    # Pre-installation checks
    Write-Log "`nPreparing for Python installation..." -Level Info
    
    # Check for running Python installers
    Write-Log "Checking for running installers..."
    $runningInstallers = Get-Process | Where-Object { $_.ProcessName -match "python.*-amd64|python.*installer" }
    if ($runningInstallers) {
        Write-Log "Found running Python installer(s). Waiting for them to finish..." -Level Warning
        Start-Sleep -Seconds 5
        $runningInstallers | Stop-Process -Force -ErrorAction SilentlyContinue
    }
    
    # Install Python
    Write-Log "`nInstalling Python $pythonVersion..." -Level Info
    Write-Log "This may take 2-3 minutes..." -Level Info
    
    try {
        # Create detailed log file for troubleshooting
        $pythonLogPath = "$env:TEMP\python_install.log"
        
        $installArgs = @(
            "/quiet",
            "InstallAllUsers=1",
            "PrependPath=1",
            "Include_pip=1",
            "Include_launcher=1",
            "Include_test=0",
            "/log", $pythonLogPath
        )
        
        Write-Log "Starting Python installer..."
        $process = Start-Process -FilePath $installerPath -ArgumentList $installArgs `
            -Wait -PassThru -NoNewWindow
        
        if ($process.ExitCode -ne 0) {
            Write-Log "Python installation failed with exit code: $($process.ExitCode)" -Level Warning
            
            # Provide specific error guidance
            switch ($process.ExitCode) {
                1603 { 
                    Write-Log "Error 1603: Fatal error during installation" -Level Warning
                    Write-Log "Common causes:" -Level Info
                    Write-Log "  - Another installer is running (check Task Manager)" -Level Info
                    Write-Log "  - Windows Installer service issue" -Level Info
                    Write-Log "  - Previous failed installation" -Level Info
                    Write-Log "`nTroubleshooting steps:" -Level Info
                    Write-Log "  1. Close all installers and try again" -Level Info
                    Write-Log "  2. Restart Windows Installer service:" -Level Info
                    Write-Log "     Restart-Service msiserver" -Level Info
                    Write-Log "  3. Check detailed log: $pythonLogPath" -Level Info
                    Write-Log "  4. Try manual install from: https://www.python.org/downloads/" -Level Info
                }
                1602 { Write-Log "Error 1602: User cancelled installation" -Level Warning }
                1605 { Write-Log "Error 1605: This action is only valid for products currently installed" -Level Warning }
                1618 { Write-Log "Error 1618: Another installation is already in progress" -Level Warning }
                default { Write-Log "Unknown error code: $($process.ExitCode)" -Level Warning }
            }
            
            # Check if installation log exists
            if (Test-Path $pythonLogPath) {
                Write-Log "`nLast 20 lines of installation log:" -Level Info
                $logContent = Get-Content $pythonLogPath -Tail 20 -ErrorAction SilentlyContinue
                $logContent | ForEach-Object { Write-Log "  $_" }
            }
            
            # Open the directory containing the downloaded installer for manual installation
            if (Test-Path $installerPath) {
                Write-Log "`nOpening folder with downloaded installer for manual installation..." -Level Info
                Write-Log "Installer location: $installerPath" -Level Info
                try {
                    # Open Windows Explorer and select the installer file
                    Start-Process "explorer.exe" -ArgumentList "/select,`"$installerPath`""
                } catch {
                    Write-Log "Could not open folder automatically" -Level Warning
                }
            }
            
            # Offer alternative installation method
            Write-Log "`nWould you like to try an alternative installation method?" -Level Warning
            Write-Log "Alternative: Use embeddable Python (portable, no Windows Installer)" -Level Info
            
            # If AutoInstallPython is set, automatically try embeddable Python
            if ($AutoInstallPython) {
                Write-Log "AutoInstallPython flag detected - automatically trying embeddable Python..." -Level Info
                return Install-PythonEmbeddable
            }
            
            if (-not $Silent) {
                $tryAlternative = Read-Host "Try embeddable Python? (Y/N)"
                if ($tryAlternative -eq "Y") {
                    Write-Log "Attempting embeddable Python installation..." -Level Info
                    # Call embeddable Python function
                    return Install-PythonEmbeddable
                }
            }
            
            return $false
        }
        
        # Refresh environment variables
        Write-Log "Refreshing environment variables..."
        $env:Path = [System.Environment]::GetEnvironmentVariable("Path", "Machine") + ";" + 
                    [System.Environment]::GetEnvironmentVariable("Path", "User")
        
        # Wait a moment for PATH to update
        Start-Sleep -Seconds 2
        
        # Verify Python installation
        Write-Log "Verifying Python installation..."
        $pythonCmd = Get-Command python -ErrorAction SilentlyContinue
        
        if ($pythonCmd) {
            $pythonVer = & python --version 2>&1
            Write-Success "Python installed successfully: $pythonVer"
            
            # Verify pip
            $pipCmd = Get-Command pip -ErrorAction SilentlyContinue
            if ($pipCmd) {
                Write-Success "pip is available"
            } else {
                Write-Log "pip not found in PATH, but should be available" -Level Warning
            }
            
            # Cleanup installer
            Write-Log "Cleaning up installer..."
            Remove-Item $installerPath -Force -ErrorAction SilentlyContinue
            
            return $true
        } else {
            Write-Log "Python installed but not found in PATH" -Level Warning
            Write-Log "Try closing and reopening PowerShell" -Level Warning
            return $false
        }
        
    } catch {
        Write-Log "Installation failed: $($_.Exception.Message)" -Level Warning
        
        # Clean up failed download
        if (Test-Path $installerPath) {
            Remove-Item -Path $installerPath -Force -ErrorAction SilentlyContinue
        }
        
        return $false
    }
}

function Install-PythonEmbeddable {
    <#
    .SYNOPSIS
        Install embeddable (portable) Python as fallback when standard installer fails
    #>
    
    Write-Log "`n=== Installing Embeddable Python 3.11.9 ===" -Level Info
    Write-Log "This is a portable version that doesn't require Windows Installer" -Level Info
    
    $pythonVersion = "3.11.9"
    $embedUrl = "https://www.python.org/ftp/python/$pythonVersion/python-$pythonVersion-embed-amd64.zip"
    $embedZip = "$env:TEMP\python-$pythonVersion-embed.zip"
    $installLocation = "C:\Python311"
    
    try {
        # Download embeddable Python
        Write-Log "Downloading embeddable Python..." -Level Info
        [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
        
        $ProgressPreference = 'SilentlyContinue'
        Invoke-WebRequest -Uri $embedUrl -OutFile $embedZip -UseBasicParsing -ErrorAction Stop
        $ProgressPreference = 'Continue'
        
        if (-not (Test-Path $embedZip)) {
            throw "Failed to download embeddable Python"
        }
        
        Write-Success "Downloaded embeddable Python"
        
        # Extract to install location
        Write-Log "Extracting to $installLocation..."
        if (Test-Path $installLocation) {
            Write-Log "Removing existing installation..." -Level Warning
            Remove-Item $installLocation -Recurse -Force -ErrorAction SilentlyContinue
        }
        
        New-Item -Path $installLocation -ItemType Directory -Force | Out-Null
        Expand-Archive -Path $embedZip -DestinationPath $installLocation -Force
        
        Write-Success "Python extracted to $installLocation"
        
        # Enable site-packages by modifying python311._pth
        Write-Log "Configuring Python..."
        $pthFile = Get-ChildItem -Path $installLocation -Filter "python*._pth" | Select-Object -First 1
        if ($pthFile) {
            $content = Get-Content $pthFile.FullName
            $content = $content -replace "^#import site", "import site"
            $content | Set-Content $pthFile.FullName
        }
        
        # Download and install pip
        Write-Log "Installing pip..."
        $getPipUrl = "https://bootstrap.pypa.io/get-pip.py"
        $getPipPath = "$installLocation\get-pip.py"
        
        Invoke-WebRequest -Uri $getPipUrl -OutFile $getPipPath -UseBasicParsing -ErrorAction Stop
        
        $pythonExe = "$installLocation\python.exe"
        & $pythonExe $getPipPath --no-warn-script-location 2>&1 | Out-Null
        
        Write-Success "pip installed"
        
        # Add to system PATH
        Write-Log "Adding Python to system PATH..."
        $currentPath = [Environment]::GetEnvironmentVariable("Path", "Machine")
        
        if ($currentPath -notlike "*$installLocation*") {
            $newPath = "$installLocation;$installLocation\Scripts;$currentPath"
            [Environment]::SetEnvironmentVariable("Path", $newPath, "Machine")
            Write-Success "Added to system PATH"
        }
        
        # Refresh current session PATH
        $env:Path = [System.Environment]::GetEnvironmentVariable("Path", "Machine") + ";" + 
                    [System.Environment]::GetEnvironmentVariable("Path", "User")
        
        # Verify installation
        Start-Sleep -Seconds 2
        Write-Log "Verifying Python installation..."
        
        try {
            $pythonVer = & python --version 2>&1
            Write-Success "Python installed successfully: $pythonVer"
            
            $pipVer = & pip --version 2>&1
            Write-Success "pip installed successfully: $pipVer"
            
            # Cleanup
            Remove-Item $embedZip -Force -ErrorAction SilentlyContinue
            Remove-Item $getPipPath -Force -ErrorAction SilentlyContinue
            
            Write-Log "`nEmbeddable Python installation completed!" -Level Info
            Write-Log "Location: $installLocation" -Level Info
            
            return $true
        }
        catch {
            Write-Log "Python installed but not accessible in current session" -Level Warning
            Write-Log "Please close and reopen PowerShell, then run deployment again" -Level Info
            return $false
        }
    }
    catch {
        Write-Log "Embeddable Python installation failed: $($_.Exception.Message)" -Level Warning
        
        # Cleanup on failure
        if (Test-Path $embedZip) {
            Remove-Item $embedZip -Force -ErrorAction SilentlyContinue
        }
        
        Write-Log "`nPlease install Python manually:" -Level Info
        Write-Log "  1. Download from: https://www.python.org/downloads/" -Level Info
        Write-Log "  2. Or use: winget install Python.Python.3.11" -Level Info
        Write-Log "  3. Or use: choco install python311" -Level Info
        
        return $false
    }
}

function New-PythonVirtualEnvironment {
    param([string]$VenvPath)
    
    Write-Log "Creating Python virtual environment..."
    
    if (Test-Path $VenvPath) {
        Write-Log "Virtual environment already exists" -Level Warning
        $overwrite = $true
        if (-not $Silent) {
            $response = Read-Host "Overwrite existing virtual environment? (Y/N)"
            $overwrite = $response -eq "Y"
        }
        
        if ($overwrite) {
            Write-Log "Removing existing virtual environment..."
            Remove-Item -Path $VenvPath -Recurse -Force
        }
        else {
            Write-Log "Using existing virtual environment"
            return $true
        }
    }
    
    try {
        & python -m venv $VenvPath
        Write-Success "Virtual environment created: $VenvPath"
        return $true
    }
    catch {
        Write-InstallError "Failed to create virtual environment" $_
        return $false
    }
}

function Install-PythonDependencies {
    param([string]$VenvPath, [string]$RequirementsFile)
    
    Write-Log "Installing Python dependencies..."
    
    $pipPath = Join-Path $VenvPath "Scripts\pip.exe"
    if (-not (Test-Path $pipPath)) {
        Write-InstallError "pip not found in virtual environment"
        return $false
    }
    
    try {
        # Upgrade pip first
        Write-Log "Upgrading pip..."
        & $pipPath install --upgrade pip | Out-Null
        
        # Install requirements
        Write-Log "Installing from: $RequirementsFile"
        & $pipPath install -r $RequirementsFile
        
        # Run pywin32 post-install script (required for Windows service support)
        Write-Log "Configuring pywin32 for Windows service support..."
        $pythonPath = Join-Path $VenvPath "Scripts\python.exe"
        $pywin32Script = Join-Path $VenvPath "Scripts\pywin32_postinstall.py"
        
        if (Test-Path $pywin32Script) {
            try {
                & $pythonPath $pywin32Script -install
                Write-Log "pywin32 configured successfully"
            }
            catch {
                Write-Log "pywin32 post-install warning: $($_.Exception.Message)" -Level Warning
                Write-Log "Service may require manual configuration" -Level Warning
            }
        }
        else {
            Write-Log "pywin32_postinstall.py not found - skipping" -Level Warning
        }
        
        Write-Success "Dependencies installed successfully"
        return $true
    }
    catch {
        Write-InstallError "Failed to install dependencies" $_
        return $false
    }
}

#endregion

#region Backup and Deployment

function Backup-ExistingInstallation {
    param([string]$BackupPath)
    
    if ($SkipBackup) {
        Write-Log "Skipping backup (SkipBackup flag set)"
        return $null
    }
    
    if (-not (Test-Path $InstallPath)) {
        Write-Log "No existing installation to backup"
        return $null
    }
    
    Write-Log "Backing up existing installation..."
    
    $timestamp = Get-Date -Format "yyyyMMdd_HHmmss"
    $backupDir = "$BackupPath\ArcVault_Backup_$timestamp"
    
    try {
        # Stop service if running
        $service = Get-Service -Name "ArcVaultAgent" -ErrorAction SilentlyContinue
        if ($service -and $service.Status -eq "Running") {
            Write-Log "Stopping ArcVault service..."
            Stop-Service -Name "ArcVaultAgent" -Force
            Start-Sleep -Seconds 2
        }
        
        # Create backup
        Write-Log "Creating backup at: $backupDir"
        Copy-Item -Path $InstallPath -Destination $backupDir -Recurse -Force
        
        # Backup data directory if exists
        if (Test-Path $DataPath) {
            $dataBackup = "$backupDir\data"
            Copy-Item -Path $DataPath -Destination $dataBackup -Recurse -Force
        }
        
        Write-Success "Backup created: $backupDir"
        return $backupDir
    }
    catch {
        Write-InstallError "Backup failed (continuing anyway)" $_
        return $null
    }
}

function Copy-ApplicationFiles {
    param([string]$Source, [string]$Destination)
    
    Write-Log "Deploying application files..."
    Write-Log "Source: $Source"
    Write-Log "Destination: $Destination"
    
    try {
        # Ensure service is stopped before file operations
        $service = Get-Service -Name "ArcVaultAgent" -ErrorAction SilentlyContinue
        if ($service -and $service.Status -eq "Running") {
            Write-Log "Stopping ArcVault service before deployment..."
            Stop-Service -Name "ArcVaultAgent" -Force -ErrorAction SilentlyContinue
            Start-Sleep -Seconds 3
        }
        
        # Remove old installation directory if it exists (backup already created)
        if (Test-Path $Destination) {
            Write-Log "Removing old installation directory..."
            Remove-Item -Path $Destination -Recurse -Force -ErrorAction Stop
            Start-Sleep -Seconds 1
        }
        
        # Create destination directory
        Write-Log "Creating installation directory..."
        New-Item -Path $Destination -ItemType Directory -Force | Out-Null
        
        # Normalize source path to full path
        $normalizedSource = (Resolve-Path $Source).Path
        if (-not $normalizedSource.EndsWith("\")) {
            $normalizedSource += "\"
        }
        
        # Copy files
        $excludePatterns = @("__pycache__", "*.pyc", ".pytest_cache", "dist", "build", ".venv")
        
        Write-Log "Copying files..."
        Get-ChildItem -Path $Source -Recurse | ForEach-Object {
            $relativePath = $_.FullName.Substring($normalizedSource.Length)
            $destPath = Join-Path $Destination $relativePath
            
            # Check if should exclude
            $exclude = $false
            foreach ($pattern in $excludePatterns) {
                if ($_.FullName -like "*$pattern*") {
                    $exclude = $true
                    break
                }
            }
            
            if (-not $exclude) {
                if ($_.PSIsContainer) {
                    if (-not (Test-Path $destPath)) {
                        New-Item -Path $destPath -ItemType Directory -Force | Out-Null
                    }
                }
                else {
                    Copy-Item -Path $_.FullName -Destination $destPath -Force
                }
            }
        }
        
        Write-Success "Application files deployed"
        return $true
    }
    catch {
        Write-Log "Failed to copy application files: $($_.Exception.Message)" -Level Warning
        Write-Log "Error at: $($_.InvocationInfo.ScriptLineNumber)" -Level Warning
        
        # Check if files are still locked
        $service = Get-Service -Name "ArcVaultAgent" -ErrorAction SilentlyContinue
        if ($service -and $service.Status -eq "Running") {
            Write-Log "ArcVault service is still running - attempting to stop..." -Level Warning
            Stop-Service -Name "ArcVaultAgent" -Force -ErrorAction SilentlyContinue
            Write-Log "Please try running the deployment again" -Level Info
        }
        
        return $false
    }
}

#endregion

#region Configuration

function Initialize-DirectoryStructure {
    Write-Log "Creating directory structure..."
    
    $directories = @(
        $InstallPath,
        $DataPath,
        "$DataPath\keys",
        "$DataPath\logs",
        "$DataPath\logs\#backup_logs",
        "$DataPath\config"
    )
    
    foreach ($dir in $directories) {
        if (-not (Test-Path $dir)) {
            New-Item -Path $dir -ItemType Directory -Force | Out-Null
            Write-Log "Created: $dir"
        }
    }
    
    Write-Success "Directory structure created"
}

function Copy-ConfigurationFiles {
    Write-Log "Deploying configuration files..."
    
    $scriptDir = Split-Path -Parent $PSCommandPath
    $configSource = Join-Path $scriptDir "config"
    
    if (-not (Test-Path $configSource)) {
        Write-Log "No default configuration files found" -Level Warning
        return
    }
    
    try {
        # Copy service config
        $serviceConfig = Join-Path $configSource "default_service_config.json"
        if (Test-Path $serviceConfig) {
            $destConfig = "$DataPath\service_config.json"
            if (-not (Test-Path $destConfig)) {
                Copy-Item -Path $serviceConfig -Destination $destConfig -Force
                Write-Log "Deployed: service_config.json"
            }
            else {
                Write-Log "service_config.json already exists, skipping" -Level Warning
            }
        }
        
        # Copy backup config
        $backupConfig = Join-Path $configSource "default_backup_config.json"
        if (Test-Path $backupConfig) {
            $destConfig = "$DataPath\backup_config.json"
            if (-not (Test-Path $destConfig)) {
                Copy-Item -Path $backupConfig -Destination $destConfig -Force
                Write-Log "Deployed: backup_config.json"
            }
        }
        
        # Copy environment file
        $envFile = Join-Path $configSource "service.env"
        if (Test-Path $envFile) {
            $destEnv = "$DataPath\.env"
            if (-not (Test-Path $destEnv)) {
                # Read and update port
                $envContent = Get-Content $envFile
                $envContent = $envContent -replace "AGENT_PORT=\d+", "AGENT_PORT=$Port"
                $envContent | Set-Content $destEnv
                Write-Log "Deployed: .env"
            }
        }
        
        Write-Success "Configuration files deployed"
    }
    catch {
        Write-InstallError "Failed to copy configuration files" $_
    }
}

function New-SSLCertificate {
    Write-Log "Generating SSL certificate..."
    
    $certPath = "$DataPath\keys\server.crt"
    $keyPath = "$DataPath\keys\server.key"
    
    if ((Test-Path $certPath) -and (Test-Path $keyPath)) {
        Write-Log "SSL certificate already exists" -Level Warning
        return $true
    }
    
    try {
        # Use Python to generate certificate
        $pythonPath = Join-Path $InstallPath "venv\Scripts\python.exe"
        $genScript = Join-Path $SourcePath "cli\cert_generator.py"
        
        if (Test-Path $genScript) {
            Write-Log "Using cert_generator.py..."
            & $pythonPath $genScript --output "$DataPath\keys" --days 3650
        }
        else {
            # Fallback: use OpenSSL if available
            Write-Log "Looking for OpenSSL..."
            $openssl = Get-Command openssl -ErrorAction SilentlyContinue
            if ($openssl) {
                & openssl req -x509 -newkey rsa:4096 -keyout $keyPath -out $certPath -days 3650 -nodes -subj "/CN=ArcVault"
            }
            else {
                Write-Log "SSL certificate generation skipped (will use HTTP)" -Level Warning
                return $true
            }
        }
        
        Write-Success "SSL certificate generated"
        return $true
    }
    catch {
        Write-Log "SSL certificate generation failed (will use HTTP)" -Level Warning
        return $true
    }
}

function New-JWTKeys {
    Write-Log "Generating JWT key pair..."
    
    $privateKeyPath = "$DataPath\keys\jwt_private.pem"
    $publicKeyPath = "$DataPath\keys\jwt_public.pem"
    
    if ((Test-Path $privateKeyPath) -and (Test-Path $publicKeyPath)) {
        Write-Log "JWT keys already exist" -Level Warning
        return $true
    }
    
    try {
        # Use Python to generate JWT keys
        $pythonPath = Join-Path $InstallPath "venv\Scripts\python.exe"
        $genScript = Join-Path $SourcePath "cli\jwt_generator.py"
        
        if (Test-Path $genScript) {
            Write-Log "Using jwt_generator.py..."
            & $pythonPath $genScript --output "$DataPath\keys"
        }
        else {
            # Fallback: use OpenSSL if available
            Write-Log "Looking for OpenSSL..."
            $openssl = Get-Command openssl -ErrorAction SilentlyContinue
            if ($openssl) {
                & openssl genrsa -out $privateKeyPath 4096
                & openssl rsa -in $privateKeyPath -pubout -out $publicKeyPath
            }
            else {
                Write-Log "JWT key generation skipped" -Level Warning
                return $true
            }
        }
        
        Write-Success "JWT keys generated"
        return $true
    }
    catch {
        Write-Log "JWT key generation failed" -Level Warning
        return $true
    }
}

#endregion

#region Service Installation

function Install-WindowsService {
    Write-Log "Installing Windows service..."
    
    if ($Development) {
        Write-Log "Development mode: Skipping service installation"
        return $true
    }
    
    try {
        # Check if service already exists
        $existingService = Get-Service -Name "ArcVaultAgent" -ErrorAction SilentlyContinue
        if ($existingService) {
            Write-Log "Service already exists, removing..." -Level Warning
            
            if ($existingService.Status -eq "Running") {
                Stop-Service -Name "ArcVaultAgent" -Force
                Start-Sleep -Seconds 2
            }
            
            # Use Python service to remove itself
            $pythonPath = Join-Path $InstallPath "venv\Scripts\python.exe"
            $servicePath = Join-Path $InstallPath "arcvault_unified.py"
            
            & $pythonPath $servicePath remove
            Start-Sleep -Seconds 2
        }
        
        # Install new service
        Write-Log "Installing service..."
        $pythonPath = Join-Path $InstallPath "venv\Scripts\python.exe"
        $servicePath = Join-Path $InstallPath "arcvault_unified.py"
        
        # Copy service config to installation directory
        Copy-Item "$DataPath\service_config.json" "$InstallPath\service_config.json" -Force
        
        & $pythonPath $servicePath install
        
        # Set service to automatic start
        Set-Service -Name "ArcVaultAgent" -StartupType Automatic
        
        Write-Success "Windows service installed"
        return $true
    }
    catch {
        Write-InstallError "Failed to install Windows service" $_
        return $false
    }
}

function Start-ArcVaultService {
    Write-Log "Starting ArcVault service..."
    
    if ($Development) {
        Write-Log "Development mode: Service not started"
        Write-Log "Run manually with: python arcvault_unified.py" -Level Info
        return $true
    }
    
    try {
        Start-Service -Name "ArcVaultAgent"
        Start-Sleep -Seconds 5
        
        $service = Get-Service -Name "ArcVaultAgent"
        if ($service.Status -eq "Running") {
            Write-Success "ArcVault service started successfully"
            return $true
        }
        else {
            Write-Log "Service status: $($service.Status)" -Level Warning
            Write-Log "Service may take additional time to start" -Level Info
            Write-Log "Check service status: Get-Service ArcVaultAgent" -Level Info
            Write-Log "Check logs at: $env:TEMP\arcvault_service.log" -Level Info
            return $false
        }
    }
    catch {
        Write-Log "Failed to start service: $($_.Exception.Message)" -Level Warning
        Write-Log "This is often normal - service may start on next boot" -Level Info
        Write-Log "Check service status: Get-Service ArcVaultAgent" -Level Info
        Write-Log "Check logs at: $env:TEMP\arcvault_service.log" -Level Info
        Write-Log "View event log: Get-EventLog -LogName Application -Source ArcVaultAgent -Newest 5" -Level Info
        return $false
    }
}

#endregion

#region Verification

function Test-Installation {
    Write-Step "Verifying Installation"
    
    $checks = @()
    
    # Check installation directory
    $checks += @{
        Name = "Installation directory"
        Test = Test-Path $InstallPath
    }
    
    # Check Python executable
    $pythonPath = Join-Path $InstallPath "venv\Scripts\python.exe"
    $checks += @{
        Name = "Python virtual environment"
        Test = Test-Path $pythonPath
    }
    
    # Check main application file
    $appPath = Join-Path $InstallPath "arcvault_unified.py"
    $checks += @{
        Name = "Application files"
        Test = Test-Path $appPath
    }
    
    # Check data directory
    $checks += @{
        Name = "Data directory"
        Test = Test-Path $DataPath
    }
    
    # Check configuration
    $configPath = "$DataPath\service_config.json"
    $checks += @{
        Name = "Service configuration"
        Test = Test-Path $configPath
    }
    
    # Check service (if not development mode)
    if (-not $Development) {
        $service = Get-Service -Name "ArcVaultAgent" -ErrorAction SilentlyContinue
        $checks += @{
            Name = "Windows service"
            Test = $null -ne $service
        }
        
        if ($service) {
            # Check service status but don't fail installation if not running
            if ($service.Status -eq "Running") {
                $checks += @{
                    Name = "Service running"
                    Test = $true
                }
            }
            else {
                Write-Log "Service status: $($service.Status) (can be started manually)" -Level Warning
            }
        }
    }
    
    # Run checks
    $allPassed = $true
    foreach ($check in $checks) {
        if ($check.Test) {
            Write-Success $check.Name
        }
        else {
            Write-Log "$($check.Name): FAILED" -Level Warning
            $allPassed = $false
        }
    }
    
    return $allPassed
}

#endregion

#region Main Deployment Flow

function Start-Deployment {
    Write-Log "`n================================================================="
    Write-Log "          ArcVault Professional Deployment v$Version          "
    Write-Log "=================================================================`n"
    
    Write-Log "Installation Parameters:"
    Write-Log "  Install Path: $InstallPath"
    Write-Log "  Data Path:    $DataPath"
    Write-Log "  Port:         $Port"
    Write-Log "  Source Path:  $SourcePath"
    Write-Log "  Development:  $Development"
    Write-Log "  Log File:     $LogPath"
    Write-Log ""
    
    try {
        # Pre-flight checks
        Write-Step "Pre-Flight Checks"
        Test-Administrator
        Test-PowerShellVersion
        
        if (-not (Test-DiskSpace -Path (Split-Path $InstallPath) -RequiredGB 2)) {
            throw "Insufficient disk space"
        }
        
        if (-not (Test-PortAvailable -TestPort $Port)) {
            Write-Log "Port $Port is in use. Installation will continue, but you may need to change the port." -Level Warning
        }
        
        if (-not $ConfigureOnly) {
            if (-not (Test-SourceFiles)) {
                throw "Source files validation failed"
            }
        }
        
        # Python environment
        if (-not $ConfigureOnly) {
            Write-Step "Python Environment"
            if (-not (Install-Python)) {
                throw "Python installation required"
            }
        }
        
        # Backup existing installation
        Write-Step "Backup"
        $backupPath = Backup-ExistingInstallation -BackupPath (Split-Path $InstallPath)
        if ($backupPath) {
            Write-Log "Backup location: $backupPath" -Level Info
        }
        
        # Create directory structure
        Write-Step "Directory Structure"
        Initialize-DirectoryStructure
        
        # Copy application files
        if (-not $ConfigureOnly) {
            Write-Step "Application Deployment"
            if (-not (Copy-ApplicationFiles -Source $SourcePath -Destination $InstallPath)) {
                throw "Failed to deploy application files"
            }
            
            # Create virtual environment
            Write-Step "Python Virtual Environment"
            $venvPath = Join-Path $InstallPath "venv"
            if (-not (New-PythonVirtualEnvironment -VenvPath $venvPath)) {
                throw "Failed to create virtual environment"
            }
            
            # Install dependencies
            $requirementsFile = Join-Path $InstallPath "requirements_unified.txt"
            if (-not (Install-PythonDependencies -VenvPath $venvPath -RequirementsFile $requirementsFile)) {
                throw "Failed to install dependencies"
            }
        }
        
        # Configuration
        Write-Step "Configuration"
        Copy-ConfigurationFiles
        New-SSLCertificate
        New-JWTKeys
        
        # Service installation
        if (-not $ConfigureOnly) {
            Write-Step "Windows Service Installation"
            if (-not (Install-WindowsService)) {
                throw "Failed to install Windows service"
            }
            
            # Start service
            Write-Step "Starting Service"
            $serviceStarted = Start-ArcVaultService
            if (-not $serviceStarted) {
                Write-Log "`nService did not start automatically, but this is OK" -Level Warning
                Write-Log "You can start it manually with: Start-Service ArcVaultAgent" -Level Info
            }
        }
        
        # Verification
        if (Test-Installation) {
            Write-Success "`nInstallation completed successfully!"
            $script:InstallationSuccess = $true
        }
        else {
            Write-Log "`nInstallation completed with warnings" -Level Warning
        }
        
        # Show next steps
        Write-Step "Next Steps"
        Write-Log "1. Verify service status: Get-Service ArcVaultAgent"
        Write-Log "2. Check service logs: Get-Content $env:TEMP\arcvault_service.log"
        Write-Log "3. Access agent API: http://localhost:$Port"
        Write-Log "4. Run verification: .\Verify-ArcVault.ps1"
        
        if ($Development) {
            Write-Log "`nDevelopment Mode:"
            Write-Log "  Start agent: cd '$InstallPath' && .\venv\Scripts\python.exe arcvault_unified.py"
        }
        
    }
    catch {
        Write-InstallError "`n[ERROR] Installation failed: $($_.Exception.Message)" $_
        
        if ($backupPath) {
            Write-Log "`nBackup available at: $backupPath" -Level Info
            Write-Log "Run rollback: .\Rollback-ArcVault.ps1 -BackupPath '$backupPath'" -Level Info
        }
        
        exit 1
    }
    finally {
        $duration = (Get-Date) - $script:StartTime
        Write-Log "`nTotal installation time: $($duration.ToString('mm\:ss'))"
        Write-Log "Log file: $LogPath"
    }
}

#endregion

# Start deployment
Start-Deployment

# Return success code
if ($script:InstallationSuccess) {
    exit 0
}
else {
    exit 1
}

