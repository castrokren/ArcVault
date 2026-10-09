@echo off
echo ========================================
echo ArcVault Agent Installer Builder
echo ========================================
echo.

REM Check if Inno Setup is installed
if not exist "C:\Program Files (x86)\Inno Setup 6\ISCC.exe" (
    echo ERROR: Inno Setup not found!
    echo.
    echo Please install Inno Setup 6 from:
    echo https://jrsoftware.org/isdl.php
    echo.
    pause
    exit /b 1
)

REM Check if executables exist
if not exist "dist\ArcVaultAgentService.exe" (
    echo ERROR: Service executable not found!
    echo Please build executables first using PyInstaller
    echo.
    pause
    exit /b 1
)

if not exist "dist\ArcVaultAgent.exe" (
    echo ERROR: Standalone executable not found!
    echo Please build executables first using PyInstaller
    echo.
    pause
    exit /b 1
)

echo Checking files...
echo   [OK] Inno Setup found
echo   [OK] ArcVaultAgentService.exe found
echo   [OK] ArcVaultAgent.exe found
echo.

echo Building installer...
echo This may take 30-60 seconds...
echo.

"C:\Program Files (x86)\Inno Setup 6\ISCC.exe" ArcVaultAgentInstaller.iss

if %errorlevel% equ 0 (
    echo.
    echo ========================================
    echo SUCCESS! Installer built successfully!
    echo ========================================
    echo.
    echo Location: installer_output\ArcVaultAgent_Setup_1.0.0.exe
    echo.
    for %%A in (installer_output\ArcVaultAgent_Setup_*.exe) do (
        echo File size: %%~zA bytes
        echo Created: %%~tA
    )
    echo.
    echo Opening output folder...
    explorer installer_output
) else (
    echo.
    echo ========================================
    echo BUILD FAILED
    echo ========================================
    echo.
    echo Error code: %errorlevel%
    echo Check the error messages above
)

echo.
pause

