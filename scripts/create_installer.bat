@echo off
setlocal ENABLEDELAYEDEXPANSION

REM Create a single-file installer using IExpress or similar
REM This creates a self-extracting archive with the deployment package

echo ========================================
echo ArcVault Single-File Installer Creator
echo ========================================
echo.

set "SCRIPT_DIR=%~dp0"
pushd "%SCRIPT_DIR%\.."

REM Check if deployment package exists
if not exist "deployment" (
    echo ERROR: Deployment package not found.
    echo Please run build_deployment_package.bat first.
    pause
    exit /b 1
)

REM Create installer directory
set "INSTALLER_DIR=installer"
if exist "%INSTALLER_DIR%" rmdir /s /q "%INSTALLER_DIR%"
mkdir "%INSTALLER_DIR%"

echo Creating single-file installer...

REM Create IExpress configuration file
(
echo [Version]
echo Class=IEXPRESS
echo SEDVersion=3
echo.
echo [Options]
echo PackagePurpose=InstallApp
echo ShowInstallProgramWindow=0
echo HideExtractAnimation=0
echo UseLongFileName=1
echo InsideCompressed=0
echo CAB_FixedSize=0
echo CAB_ResvCodeSigning=0
echo RebootMode=N
echo InstallPrompt=%InstallPrompt%
echo DisplayLicense=%DisplayLicense%
echo FinishMessage=%FinishMessage%
echo TargetName=%TargetName%
echo FriendlyName=%FriendlyName%
echo AppLaunched=%AppLaunched%
echo PostInstallCmd=%PostInstallCmd%
echo AdminQuietInstCmd=%AdminQuietInstCmd%
echo UserQuietInstCmd=%UserQuietInstCmd%
echo SourceFiles=SourceFiles
echo.
echo [Strings]
echo InstallPrompt=ArcVault will be installed on your system. Do you want to continue?
echo DisplayLicense=
echo FinishMessage=ArcVault has been installed successfully. You can now run the setup wizard.
echo TargetName=ArcVault-Installer.exe
echo FriendlyName=ArcVault Backup System Installer
echo AppLaunched=ArcVault-Deploy.bat
echo PostInstallCmd=
echo AdminQuietInstCmd=
echo UserQuietInstCmd=
echo FILE0="ArcVault-Deploy.bat"
echo FILE1="install.bat"
echo FILE2="setup.bat"
echo FILE3="service-manager.bat"
echo FILE4="ArcVaultKeyGen.exe"
echo FILE5="agent\ArcVaultAgent.exe"
echo FILE6="agent\ArcVaultAgentService.exe"
echo FILE7="controller\ArcVaultController.exe"
echo FILE8="config\development.env"
echo FILE9="config\staging.env"
echo FILE10="config\production.env"
echo FILE11="README.txt"
echo.
echo [SourceFiles]
echo SourceFiles0=deployment\
echo.
echo [SourceFiles0]
echo %%FILE0%%=
echo %%FILE1%%=
echo %%FILE2%%=
echo %%FILE3%%=
echo %%FILE4%%=
echo %%FILE5%%=
echo %%FILE6%%=
echo %%FILE7%%=
echo %%FILE8%%=
echo %%FILE9%%=
echo %%FILE10%%=
echo %%FILE11%%=
) > "%INSTALLER_DIR%\ArcVault.sed"

REM Copy deployment files to installer directory
echo Copying deployment files...
xcopy "deployment\*" "%INSTALLER_DIR%\" /E /I /Y

REM Create IExpress installer
echo Building installer...
iexpress /N "%INSTALLER_DIR%\ArcVault.sed"

if %ERRORLEVEL% NEQ 0 (
    echo ERROR: IExpress failed. Trying alternative method...
    
    REM Alternative: Create a simple batch installer
    (
    echo @echo off
    echo setlocal ENABLEDELAYEDEXPANSION
    echo.
    echo echo ========================================
    echo echo ArcVault Installer
    echo echo ========================================
    echo echo.
    echo.
    echo REM Extract files to temp directory
    echo set TEMP_DIR=%%TEMP%%\ArcVault_Install
    echo if exist "%%TEMP_DIR%%" rmdir /s /q "%%TEMP_DIR%%"
    echo mkdir "%%TEMP_DIR%%"
    echo.
    echo echo Extracting ArcVault files...
    echo.
    echo REM Copy all files from installer directory
    echo for /r "%%~dp0" %%f in ^(*.*^) do ^(
    echo     set "relpath=%%f"
    echo     set "relpath=!relpath:%%~dp0=!"
    echo     set "target=%%TEMP_DIR%%!relpath!"
    echo     for %%d in ^("!target!"^) do mkdir "%%~dpd" 2^>nul
    echo     copy "%%f" "!target!" ^>nul
    echo ^)
    echo.
    echo echo Files extracted to: %%TEMP_DIR%%
    echo echo.
    echo echo Starting ArcVault setup...
    echo cd /d "%%TEMP_DIR%%"
    echo call ArcVault-Deploy.bat
    ) > "%INSTALLER_DIR%\ArcVault-Installer.bat"
    
    echo Created batch installer: %INSTALLER_DIR%\ArcVault-Installer.bat
) else (
    echo IExpress installer created successfully!
)

REM Create a simple ZIP alternative
echo Creating ZIP package...
powershell -Command "Compress-Archive -Path 'deployment\*' -DestinationPath '%INSTALLER_DIR%\ArcVault-Deployment.zip' -Force"

echo.
echo ========================================
echo Installer Creation Complete!
echo ========================================
echo.
echo Created in: %INSTALLER_DIR%\
echo.
echo Files:
if exist "%INSTALLER_DIR%\ArcVault-Installer.exe" (
    echo - ArcVault-Installer.exe (IExpress installer)
)
if exist "%INSTALLER_DIR%\ArcVault-Installer.bat" (
    echo - ArcVault-Installer.bat (Batch installer)
)
echo - ArcVault-Deployment.zip (ZIP package)
echo.
echo Deployment options:
echo 1. Use ArcVault-Installer.exe for single-file deployment
echo 2. Use ArcVault-Deployment.zip for manual extraction
echo 3. Copy the entire deployment folder directly
echo.

popd
endlocal
