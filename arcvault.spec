# -*- mode: python ; coding: utf-8 -*-
block_cipher = None

a = Analysis(
    ['installer/windows/arcvault_installer.py'],
    pathex=[],
    binaries=[
        ('installer/windows/dist/coordinator.exe', '.'),
        ('installer/windows/dist/agent.exe', '.'),
    ],
    # No datas: the installer reads nothing from installer/windows at runtime
    # (icon is base64-embedded; coordinator.exe/agent.exe come from binaries=
    # above). Bundling the whole folder swept in stale dist/build artifacts and
    # loose exe copies, bloating the installer ~150MB. See git log.
    datas=[],
    hiddenimports=['tkinter'],
    hookspath=[],
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
    name='ArcVault-Setup-0.6.0-windows-amd64',
    debug=False,
    bootloader_ignore_signals=False,
    strip=False,
    upx=True,
    upx_exclude=[],
    runtime_tmpdir=None,
    console=False,
    uac_admin=True,
    icon='installer/windows/icon.ico',
    disable_windowed_traceback=False,
    target_arch=None,
    codesign_identity=None,
    entitlements_file=None,
)
