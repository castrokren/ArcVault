# Build Windows Installer (.exe)

Complete step-by-step guide to build `ArcVault-Setup-1.1.0-windows-amd64.exe`

**Time required:** 45 minutes  
**Difficulty:** Intermediate  
**Platform:** Windows 10+ or Windows 11

---

## Prerequisites

### Option A: Native Windows (Recommended)
```powershell
# 1. Install Go
# Download from: https://go.dev/dl/go1.25.0.windows-amd64.msi
# Run installer, add to PATH

# 2. Install NSIS
# Download from: https://sourceforge.net/projects/nsis/files/NSIS%203/3.09/nsis-3.09-installer.exe
# Run installer (adds to PATH automatically)

# 3. Install Git (if not already)
# Download from: https://git-scm.com/download/win

# Verify installation
go version          # Should show: go version go1.25.0 windows/amd64
makensis -VERSION   # Should show: NSIS 3.09
```

### Option B: Windows + WSL2 (Also Works)
```bash
# In PowerShell (admin):
wsl --install

# In WSL terminal:
sudo apt-get update
sudo apt-get install -y golang-go nsis make git

# Verify
go version
makensis -VERSION
```

---

## Build Steps

### Step 1: Clone Repository
```powershell
cd C:\Projects
git clone https://github.com/castrokren/ArcVault.git
cd ArcVault
```

### Step 2: Verify Project Structure
```powershell
# Check that these files exist:
Test-Path "installer\windows\arcvault.nsi"           # Should be True
Test-Path "cmd\setup\main.go"                        # Should be True
Test-Path "coordinator\main.go"                      # Should be True
Test-Path "agent\main.go"                            # Should be True
```

### Step 3: Build Go Binaries
```powershell
# Navigate to project root
cd C:\Projects\ArcVault

# Build coordinator
go build -o coordinator.exe .\coordinator
Write-Host "✓ Built: coordinator.exe"

# Build agent
go build -o agent.exe .\agent
Write-Host "✓ Built: agent.exe"

# Build setup wizard
go build -o cmd\setup\arcvault-setup.exe .\cmd\setup
Write-Host "✓ Built: cmd\setup\arcvault-setup.exe"

# Verify binaries exist
ls *.exe
```

**Expected output:**
```
    Directory: C:\Projects\ArcVault

Mode                 LastWriteTime         Length Name
----                 -------------         ------ ----
-a---          5/23/2026   2:45 PM      12345678 agent.exe
-a---          5/23/2026   2:45 PM      12345678 coordinator.exe
```

### Step 4: Build Windows Installer (NSIS)
```powershell
# Compile NSIS script to .exe
makensis /V4 installer\windows\arcvault.nsi

# Check if build succeeded
if ($LASTEXITCODE -eq 0) {
    Write-Host "✓ NSIS compilation successful"
} else {
    Write-Host "✗ NSIS compilation failed with exit code: $LASTEXITCODE"
    exit 1
}
```

**What's happening:**
- makensis reads `installer\windows\arcvault.nsi`
- Compiles script into executable Windows installer
- Creates `ArcVault-Setup.exe` in current directory
- `/V4` flag shows verbose output

### Step 5: Rename Installer with Version
```powershell
# Rename with version number
Move-Item -Path "ArcVault-Setup.exe" `
          -NewName "ArcVault-Setup-1.1.0-windows-amd64.exe" `
          -Force

Write-Host "✓ Created: ArcVault-Setup-1.1.0-windows-amd64.exe"

# Verify file exists and get size
ls -Name "ArcVault-Setup-*.exe"
ls "ArcVault-Setup-*.exe" | Select-Object Length
```

**Expected output:**
```
ArcVault-Setup-1.1.0-windows-amd64.exe
       Length
       ------
     5234567
```

### Step 6: Verify Installer
```powershell
# Check file properties
$installer = "ArcVault-Setup-1.1.0-windows-amd64.exe"

if (Test-Path $installer) {
    $size = (Get-Item $installer).Length
    $sizeKB = [math]::Round($size / 1024, 2)
    
    Write-Host "✓ Installer created successfully"
    Write-Host "  File: $installer"
    Write-Host "  Size: $sizeKB KB"
    Write-Host "  Location: $(Get-Item $installer | Select-Object -ExpandProperty FullName)"
} else {
    Write-Host "✗ Installer not found!"
    exit 1
}
```

---

## Testing the Installer

### Test 1: Run Installer (GUI Test)
```powershell
# Double-click the installer to open GUI
.\ArcVault-Setup-1.1.0-windows-amd64.exe
```

**Expected behavior:**
1. Modern UI 2 welcome screen appears
2. Component selection dialog (Coordinator / Agent / Both)
3. Install path dialog
4. Configuration wizard (username, password, port, etc.)
5. Review summary
6. Installation progress
7. Dashboard opens in browser

### Test 2: Silent Installation (Scripted Test)
```powershell
# Install coordinator in silent mode
.\ArcVault-Setup-1.1.0-windows-amd64.exe /S /ComponentType=Coordinator

# Wait for installation
Start-Sleep -Seconds 10

# Verify service is running
Get-Service -Name "ArcVault*"
```

### Test 3: Verify Service
```powershell
# Check if service was installed
sc query arcvault-coordinator

# Expected output: RUNNING
```

### Test 4: Test Uninstall
```powershell
# Uninstall via Add/Remove Programs
Get-WmiObject -Class Win32_Product -Filter "Name LIKE '%ArcVault%'" | 
    ForEach-Object { $_.Uninstall() }

# Or manually
wmic product where name="ArcVault" call uninstall /nointeractive

# Verify uninstall
Get-Service -Name "ArcVault*" -ErrorAction SilentlyContinue
# Should return nothing
```

---

## Troubleshooting

### Error: "makensis: command not found"
**Solution:**
```powershell
# Check if NSIS is installed
Test-Path "C:\Program Files (x86)\NSIS\makensis.exe"

# If not found, install NSIS:
# Download: https://sourceforge.net/projects/nsis/files/NSIS%203/3.09/
# Run: nsis-3.09-installer.exe

# Add to PATH manually if needed:
$env:PATH += ";C:\Program Files (x86)\NSIS"
```

### Error: "go: command not found"
**Solution:**
```powershell
# Check if Go is installed
Test-Path "C:\Program Files\Go\bin\go.exe"

# If not found, install Go:
# Download: https://go.dev/dl/go1.25.0.windows-amd64.msi
# Run installer

# Or add to PATH if installed elsewhere:
$env:PATH += ";C:\Program Files\Go\bin"
```

### NSIS Build Fails with Error
**Solution:**
```powershell
# Run NSIS with verbose output to see the issue
makensis /V4 installer\windows\arcvault.nsi

# Common issues:
# - Missing icon file: installer\windows\installer.ico
# - Missing welcome bitmap: installer\windows\welcome.bmp
# - Script syntax error in arcvault.nsi

# Check script syntax
notepad installer\windows\arcvault.nsi
```

### Installer Size Too Large
**If .exe is > 100MB:**
```powershell
# Check what's included
# The binary should be ~15-20MB
# If larger, check for embedded files
```

---

## Success Criteria

✅ **You've successfully built the Windows installer when:**

1. **File exists:** `ArcVault-Setup-1.1.0-windows-amd64.exe` (5-10 MB)
2. **File is executable:** Can double-click and see Modern UI dialog
3. **Installation works:** Wizard completes without errors
4. **Service installs:** `sc query arcvault-coordinator` shows RUNNING
5. **Dashboard opens:** Browser automatically opens to `http://localhost:8080`
6. **Uninstall works:** Service stops, files removed, clean registry

---

## Next Steps

1. **Test installer on clean Windows VM** (if possible)
2. **Commit to git:**
   ```powershell
   git add installer/ cmd/setup/ *.exe
   git commit -m "Phase 18: Windows installer built and tested"
   git push
   ```
3. **Build macOS installer** (requires macOS machine - see BUILD_MACOS_INSTALLER.md)
4. **Build Linux installers** (see BUILD_LINUX_INSTALLERS.md)

---

## Reference: NSIS Script Details

The `installer\windows\arcvault.nsi` script:
- Uses Modern UI 2 theme (professional look)
- Supports component selection (Coordinator/Agent/Both)
- Copies binaries to install directory
- Registers Windows service via `coordinator install-service`
- Opens browser to dashboard post-install
- Creates Add/Remove Programs entry
- Includes uninstaller that stops services

Key sections in script:
```nsi
Section "Coordinator" SEC_COORDINATOR
  SetOutPath "$INSTDIR"
  File "coordinator.exe"
  File "arcvault-setup.exe"
  ExecWait '"$INSTDIR\arcvault-setup.exe"'
  ExecWait '"$INSTDIR\coordinator.exe" install-service'
SectionEnd
```

---

**Windows Installer Build Complete!** 🎉

Next: Build macOS (.pkg) or Linux (.deb/.rpm) installers
