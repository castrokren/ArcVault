# Build Windows Installer (.exe) Using Python

Complete guide to compile the Python installer to a Windows .exe without NSIS

**Time required:** 30 minutes  
**Difficulty:** Easy  
**Platform:** Windows 10+ (Python 3.8+)

---

## Why Python Instead of NSIS?

✅ **Advantages:**
- No external tools needed (just Python)
- Cross-platform development (write on any OS, test anywhere)
- Easy to modify installer behavior
- Native Windows GUI (tkinter)
- Smaller file size
- No NSIS learning curve
- Full control over installation flow

---

## Prerequisites

### Install Python 3.8+
```powershell
# Download from https://www.python.org/downloads/
# Or via Chocolatey:
choco install python

# Verify
python --version
# Expected: Python 3.x.x
```

### Install PyInstaller
```powershell
# Install PyInstaller (compiles Python to .exe)
pip install pyinstaller

# Verify
pyinstaller --version
# Expected: x.x or newer
```

### Install Go (for building binaries)
```powershell
# Download from https://go.dev/dl/
# Or via Chocolatey:
choco install golang

# Verify
go version
# Expected: go version go1.25.0 windows/amd64
```

---

## Build Steps

### Step 1: Clone Repository
```powershell
cd C:\Projects
git clone https://github.com/castrokren/ArcVault.git
cd ArcVault
```

### Step 2: Verify Files Exist
```powershell
# Check Python installer exists
Test-Path "installer\windows\arcvault_installer.py"  # Should be True

# Check Go source exists
Test-Path "coordinator\main.go"                        # Should be True
Test-Path "agent\main.go"                             # Should be True
Test-Path "cmd\setup\main.go"                         # Should be True
```

### Step 3: Build Go Binaries
```powershell
# Create dist folder
New-Item -ItemType Directory -Path "dist" -Force

# Build coordinator executable
go build -o dist\coordinator.exe .\coordinator
Write-Host "✓ Built: coordinator.exe"

# Build agent executable
go build -o dist\agent.exe .\agent
Write-Host "✓ Built: agent.exe"

# Build setup wizard executable
go build -o dist\arcvault-setup.exe .\cmd\setup
Write-Host "✓ Built: arcvault-setup.exe"

# Verify binaries
ls dist\*.exe
```

**Expected output:**
```
    Directory: C:\Projects\ArcVault\dist

Mode                 LastWriteTime         Length Name
----                 -------------         ------ ----
-a---          5/23/2026   3:15 PM      12345678 agent.exe
-a---          5/23/2026   3:15 PM      12345678 arcvault-setup.exe
-a---          5/23/2026   3:15 PM      12345678 coordinator.exe
```

### Step 4: Create Build Configuration
```powershell
# Create a build spec file for PyInstaller
$spec_content = @'
# -*- mode: python ; coding: utf-8 -*-
block_cipher = None

a = Analysis(
    ['installer\\windows\\arcvault_installer.py'],
    pathex=[],
    binaries=[
        ('dist\\coordinator.exe', '.'),
        ('dist\\agent.exe', '.'),
        ('dist\\arcvault-setup.exe', '.'),
    ],
    datas=[
        ('installer\\windows\\', 'installer\\windows'),
    ],
    hiddenimports=['tkinter'],
    hookspath=[],
    runtime_hooks=[],
    excludedimports=[],
    win_no_prefer_redirects=False,
    win_private_assemblies=False,
    cipher=block_cipher,
    noarchive=False,
)

pyz = PYZ(a.pure, a.zipped_data, cipher=block_cipher)

exe = EXE(
    pyz,
    a.scripts,
    a.binaries,
    a.zipfiles,
    a.datas,
    [],
    name='ArcVault-Setup-1.1.0-windows-amd64',
    debug=False,
    bootloader_ignore_signals=False,
    strip=False,
    upx=True,
    upx_exclude=[],
    runtime_tmpdir=None,
    console=False,
    disable_windowed_traceback=False,
    target_arch=None,
    codesign_identity=None,
    entitlements_file=None,
    icon='installer\\windows\\installer.ico',
)
'@

$spec_content | Out-File -FilePath "arcvault.spec" -Encoding UTF8
Write-Host "✓ Created: arcvault.spec"
```

### Step 5: Compile to .exe
```powershell
# Build the .exe from Python code
pyinstaller arcvault.spec --onefile

# This creates:
# - A "build" folder (temporary files)
# - A "dist" folder with ArcVault-Setup-1.1.0-windows-amd64.exe

Write-Host "✓ Compilation complete!"

# Verify the .exe was created
ls dist\ArcVault-Setup-*.exe | Select-Object Name, Length

# Expected:
# ArcVault-Setup-1.1.0-windows-amd64.exe  (30-50 MB)
```

### Step 6: Test the Installer
```powershell
# Run the compiled installer
.\dist\ArcVault-Setup-1.1.0-windows-amd64.exe
```

**Expected behavior:**
1. Window titled "ArcVault 1.1.0 Setup Wizard" appears
2. Welcome screen with Next button
3. Component selection (Coordinator/Agent/Both)
4. Configuration forms
5. Review screen
6. Installation progress
7. Success message and browser opens

---

## What the Python Installer Does

The `arcvault_installer.py` script:

✅ **Multi-step wizard:**
1. Welcome screen
2. Component selection
3. Coordinator config (port, username, password)
4. Agent config (URL, agent ID, token)
5. Review configuration
6. Install services

✅ **Features:**
- Native Windows GUI (tkinter)
- Input validation
- Auto-generated tokens
- Service registration
- Browser auto-launch
- Error handling

---

## Customizing the Installer

### Change the Window Icon
```powershell
# Create or add your icon file
# Place at: installer\windows\installer.ico

# Or download a free icon and convert:
# https://convertio.co/png-ico/

# The spec file already references it:
# icon='installer\\windows\\installer.ico',
```

### Modify the Welcome Text
```python
# Edit installer\windows\arcvault_installer.py

# Find this section:
description = ttk.Label(frame,
    text="User-friendly backup orchestration system\n\n"
         "This wizard will guide you through installing ArcVault\n"
         "on your Windows machine.",

# Change to your custom text
```

### Add More Configuration Options
```python
# Add new fields in show_coordinator_config() or show_agent_config()

# Example: Add an email field
email_frame = ttk.Frame(frame)
email_frame.pack(fill=tk.X, pady=10)
ttk.Label(email_frame, text="Email:").pack(side=tk.LEFT)
email_var = ttk.Entry(email_frame, width=30)
email_var.pack(side=tk.LEFT, padx=10)
```

---

## Troubleshooting

### Error: "Python not found"
**Solution:**
```powershell
# Check Python is installed
python --version

# If not found, install from https://www.python.org/
# Make sure to check "Add Python to PATH" during installation
```

### Error: "PyInstaller not found"
**Solution:**
```powershell
# Install PyInstaller
pip install pyinstaller

# Verify
pyinstaller --version
```

### Error: "Go binaries not found"
**Solution:**
```powershell
# Make sure you built the Go binaries
go build -o dist\coordinator.exe .\coordinator
go build -o dist\agent.exe .\agent
go build -o dist\arcvault-setup.exe .\cmd\setup

# Verify they exist
ls dist\*.exe
```

### .exe is very large (>100MB)
**Solution:**
```powershell
# This is normal for PyInstaller single-file .exe
# It includes Python runtime + all dependencies

# To reduce size, use --onedir instead of --onefile:
pyinstaller arcvault.spec --onedir

# Results in:
# dist/ArcVault-Setup-1.1.0-windows-amd64/
#   ├── ArcVault-Setup-1.1.0-windows-amd64.exe (launcher)
#   ├── python.exe
#   ├── coordinator.exe
#   └── ... (all dependencies)
```

### GUI not appearing when .exe runs
**Solution:**
```powershell
# Make sure console is disabled in spec file:
exe = EXE(
    ...
    console=False,  # This must be False
    ...
)

# Rebuild:
pyinstaller arcvault.spec --onefile
```

---

## Advanced: Multi-file vs Single-file

### Single-file (--onefile) - Current approach
```powershell
pyinstaller arcvault.spec --onefile

# Results: ArcVault-Setup-1.1.0-windows-amd64.exe (50-80 MB)
# Pros: One file to distribute
# Cons: Slower startup, larger file
```

### Multi-file (--onedir) - Smaller, faster
```powershell
pyinstaller arcvault.spec --onedir

# Results: ArcVault-Setup-1.1.0-windows-amd64\
#   ├── ArcVault-Setup-1.1.0-windows-amd64.exe
#   ├── coordinator.exe
#   ├── agent.exe
#   └── ... (libraries)
# Pros: Smaller initial exe, faster startup
# Cons: Multiple files to distribute
```

---

## Comparison: Python vs NSIS

| Feature | Python (PyInstaller) | NSIS |
|---------|----------------------|------|
| **Installation** | pip install pyinstaller | Download & install |
| **Learning curve** | Easy (Python) | Steep (NSIS language) |
| **File size** | 50-80 MB | 5-10 MB |
| **Development time** | 30 minutes | 2+ hours |
| **Customization** | Very easy (Python) | Complex (NSIS script) |
| **Cross-platform dev** | Yes | Windows only |
| **GUI customization** | Easy (tkinter) | Complex (MUI2) |

**For ArcVault: Python is better** ✅

---

## Next Steps

1. **Follow the build steps above** (5 minutes)
2. **Test the .exe** (5 minutes)
3. **Customize if needed** (optional, 10 minutes)
4. **Commit to git:**
   ```powershell
   git add installer/windows/arcvault_installer.py
   git add arcvault.spec
   git commit -m "Phase 18: Python-based Windows installer (PyInstaller)"
   git push
   ```
5. **Build release .exe** for distribution

---

## Success Criteria

✅ **You've successfully built the Windows installer when:**

1. **File exists:** `dist\ArcVault-Setup-1.1.0-windows-amd64.exe` (50-80 MB)
2. **Exe launches:** Double-click opens GUI wizard
3. **Wizard works:** All screens display correctly
4. **Installation works:** Configuration saves without errors
5. **Services install:** Services register in Windows
6. **Dashboard opens:** Browser launches to localhost:8080
7. **No errors:** No exceptions or crashes

---

## Complete Build Command (One-liner)

```powershell
# Everything in sequence:
go build -o dist\coordinator.exe .\coordinator; `
go build -o dist\agent.exe .\agent; `
go build -o dist\arcvault-setup.exe .\cmd\setup; `
pyinstaller installer\windows\arcvault.spec --onefile; `
ls dist\ArcVault-Setup-*.exe | Select-Object Name, Length
```

**Total time:** ~5 minutes for full build

---

**Windows Installer Build Complete!** 🎉

You now have a native Windows .exe installer built entirely from Python, with:
- ✅ Native GUI using tkinter
- ✅ Multi-step configuration wizard
- ✅ Full control over installation flow
- ✅ Easy to customize and maintain
- ✅ Works on Windows 10, 11, Server editions

Ready to distribute or release! 🚀
