@echo off
echo Testing ArcVault PyQt5 GUI...
echo.

REM Test if the executable exists
if not exist "installers-exe\ArcVault-GUI-PyQt5.exe" (
    echo ERROR: ArcVault-GUI-PyQt5.exe not found!
    pause
    exit /b 1
)

echo File found: installers-exe\ArcVault-GUI-PyQt5.exe
echo Size: 
powershell -Command "(Get-Item 'installers-exe\ArcVault-GUI-PyQt5.exe').Length / 1MB"
echo MB

echo.
echo Launching GUI...
start "" "installers-exe\ArcVault-GUI-PyQt5.exe"

echo.
echo GUI launched successfully!
echo If the GUI doesn't appear, check for error messages.
echo.
pause
