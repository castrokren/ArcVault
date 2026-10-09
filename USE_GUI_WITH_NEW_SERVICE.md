# Using Your PyQt5 GUI with the Improved Service

## ✅ What We Just Did

1. **Updated GUI Code** - Modified `scripts/arcvault_gui_pyqt5.py` to find the new service executable
2. **Deployed Service** - Copied improved `ArcVaultAgentService.exe` to `deployment-client/agent/`
3. **Added Config** - Placed `service_config.json` in deployment folder

## 🚀 Quick Start - Use Existing GUI

### Option 1: Test with Existing GUI (Fast!)

Your existing GUI at `installers-exe/ArcVault-GUI-PyQt5.exe` can now find and use the improved service!

**Steps:**
1. **Stop current service** (if running):
   ```powershell
   sc.exe stop ArcVaultAgent
   ```

2. **Run the GUI**:
   - Double-click: `installers-exe\ArcVault-GUI-PyQt5.exe`
   - OR from PowerShell:
   ```powershell
   cd C:\Users\castrk05\Py_stuff\ArcVault
   .\installers-exe\ArcVault-GUI-PyQt5.exe
   ```

3. **Go to "💻 Client Manager" tab**

4. **Click "Install Service"** - It will now use the improved service!

5. **Click "Start Service"**

6. **Click "Check Service Status"** - Verify it's running

7. **Click "Test Agent Connection"** - Should work!

---

## 🏗️ Option 2: Rebuild GUI (Recommended for Distribution)

If you want to distribute the GUI with the latest changes embedded:

### Step 1: Install PyInstaller (if not installed)
```powershell
pip install pyinstaller pyqt5
```

### Step 2: Build the GUI
```powershell
cd C:\Users\castrk05\Py_stuff\ArcVault

# Build GUI executable
pyinstaller --name "ArcVault-GUI-PyQt5" --windowed --onefile scripts\arcvault_gui_pyqt5.py
```

### Step 3: Find Your New GUI
- Location: `dist\ArcVault-GUI-PyQt5.exe`
- This includes the updated paths

---

## 📋 What the GUI Can Do

### Service Management Tab Features:

**Service Actions:**
- ✅ Install Service (uses improved service with logging!)
- ✅ Start Service
- ✅ Stop Service
- ✅ Uninstall Service
- ✅ Check Service Status
- ✅ Test Agent Connection

**Additional Features:**
- 🔐 SSL Certificate Generation
- 🔑 JWT Key Generation
- ⚙️ Agent Configuration
- 📊 Real-time logging and status

---

## 🎯 What's Different with the Improved Service?

When you install via the GUI, you get:

### ✅ New Features:
1. **Comprehensive Logging** → `C:\Windows\Temp\arcvault_service.log`
2. **Config File Support** → Easy settings via JSON
3. **Error Handling** → Full stack traces
4. **Thread Monitoring** → Detects crashes
5. **Heartbeat** → Logs every minute

### 📝 View Logs from GUI:
After installing and starting the service:
1. Click **"Check Service Status"** - See if it's running
2. Open: `C:\Windows\Temp\arcvault_service.log` manually
3. Or add a "View Logs" button to the GUI (see enhancement below)

---

## 🔧 Optional: Add "View Logs" Button to GUI

If you want to enhance the GUI to show service logs directly:

### Quick Enhancement:
Add this to the Service Management section in `scripts/arcvault_gui_pyqt5.py`:

```python
# Add after line 625 (after "Test Agent Connection" button)
view_logs_btn = QPushButton("View Service Logs")
view_logs_btn.clicked.connect(self.view_service_logs)
service_layout.addWidget(view_logs_btn, 3, 0)

# Add this method after test_agent_connection (around line 985)
def view_service_logs(self):
    """Open service logs in notepad"""
    import subprocess
    log_paths = [
        "C:\\Windows\\Temp\\arcvault_service.log",
        os.path.expandvars("%TEMP%\\arcvault_service.log")
    ]
    
    for log_path in log_paths:
        if os.path.exists(log_path):
            subprocess.Popen(["notepad.exe", log_path])
            self.log_message(self.installer_log, f"Opening log file: {log_path}")
            return
    
    self.log_message(self.installer_log, "No log file found")
    QMessageBox.warning(self, "Not Found", "Service log file not found")
```

---

## 🧪 Testing Checklist

### After Installing via GUI:

- [ ] Service installs successfully
- [ ] Service starts and shows "RUNNING" status
- [ ] Agent responds on http://localhost:8443
- [ ] Log file exists at `C:\Windows\Temp\arcvault_service.log`
- [ ] Log shows detailed startup information
- [ ] Config file is being read correctly
- [ ] Service auto-starts after reboot

### Quick Test Commands:
```powershell
# Check service
sc.exe query ArcVaultAgent

# Test connection
Invoke-WebRequest http://localhost:8443/docs

# View logs
notepad C:\Windows\Temp\arcvault_service.log
```

---

## 🎨 Your Complete Solution

You now have a **unified solution**:

```
ArcVault-GUI-PyQt5.exe (Your existing GUI)
├── 🔐 SSL Certificate Generator
├── 🔑 JWT Key Generator  
└── 💻 Client Manager
    ├── Install Service (NEW: Improved Service!)
    ├── Start/Stop Service
    ├── Check Status
    └── Test Connection
```

**Benefits:**
- ✅ One GUI for everything
- ✅ Beautiful PyQt5 interface
- ✅ Service with comprehensive logging
- ✅ Easy configuration
- ✅ All in one place!

---

## 🚀 Distribution

To distribute your GUI to other machines:

### Files to Include:
1. `ArcVault-GUI-PyQt5.exe` - Main GUI (from `installers-exe/` or rebuild from `dist/`)
2. `deployment-client/agent/ArcVaultAgentService.exe` - Service executable
3. `deployment-client/agent/service_config.json` - Configuration

### OR - Simple Option:
Just distribute `ArcVault-GUI-PyQt5.exe` if you rebuild it with PyInstaller - it will know where to find the service!

---

## 💡 Pro Tips

1. **Keep GUI running while managing service** - The GUI doesn't need to close after installing the service
2. **Use Test Connection** - Verify service is responding before continuing
3. **Check Status regularly** - The improved service includes health monitoring
4. **View logs for troubleshooting** - All errors are now logged with full details

---

## 🎉 Success!

You've **married** your beautiful PyQt5 GUI with the new improved Windows Service! Best of both worlds:

- ✅ Professional GUI interface
- ✅ Robust service with comprehensive logging
- ✅ Easy management and configuration
- ✅ All-in-one solution

**Your GUI is now the complete ArcVault management tool!** 🚀

