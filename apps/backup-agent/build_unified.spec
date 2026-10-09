# -*- mode: python ; coding: utf-8 -*-
"""
PyInstaller spec file for ArcVault Unified Application

Builds one executable that includes:
- GUI (PyQt5)
- Windows Service
- CLI tools (cert/JWT generation)
- FastAPI agent service

Usage:
    pyinstaller build_unified.spec
"""

block_cipher = None

a = Analysis(
    ['arcvault_unified.py'],
    pathex=[],
    binaries=[],
    datas=[
        # Include agent app and dependencies
        ('agent', 'agent'),
    ],
    hiddenimports=[
        # PyQt5 modules
        'PyQt5.QtCore',
        'PyQt5.QtGui',
        'PyQt5.QtWidgets',
        
        # Windows service modules
        'win32service',
        'win32serviceutil',
        'win32event',
        'servicemanager',
        'win32timezone',
        
        # Cryptography
        'cryptography',
        'cryptography.hazmat.primitives',
        'cryptography.hazmat.primitives.asymmetric',
        'cryptography.hazmat.primitives.asymmetric.rsa',
        'cryptography.hazmat.primitives.serialization',
        'cryptography.x509',
        
        # FastAPI/Agent
        'uvicorn',
        'fastapi',
        'starlette',
        'pydantic',
        'jwt',
        
        # GUI/Service/CLI modules
        'gui.main_window',
        'service.win_service',
        'cli.cert_generator',
        'cli.jwt_generator',
        
        # Agent modules
        'agent.app',
        'agent.engine',
        'agent.engine.config',
        'agent.engine.runner',
    ],
    hookspath=[],
    hooksconfig={},
    runtime_hooks=[],
    excludes=[],
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
    name='ArcVault',
    debug=False,
    bootloader_ignore_signals=False,
    strip=False,
    upx=True,
    upx_exclude=[],
    runtime_tmpdir=None,
    console=False,  # No console window for GUI mode
    disable_windowed_traceback=False,
    argv_emulation=False,
    target_arch=None,
    codesign_identity=None,
    entitlements_file=None,
    icon=None,  # Add icon file path if you have one
    uac_admin=False,  # Request admin only when needed, not on startup
    uac_uiaccess=False,
)

