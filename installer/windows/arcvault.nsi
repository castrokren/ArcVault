; ArcVault Installer (NSIS)
; Copies binaries and launches the setup wizard.
; All configuration, service install, and autostart is handled by arcvault-setup.exe.

!include "MUI2.nsh"

; Product info
!define PRODUCT_NAME "ArcVault"
!define PRODUCT_VERSION "0.2.1"
!define PRODUCT_PUBLISHER "ArcVault Team"
!define PRODUCT_WEB_SITE "https://github.com/castrokren/ArcVault"
!define PRODUCT_UNINST_KEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_NAME}"

; Installer attributes
Name "${PRODUCT_NAME} ${PRODUCT_VERSION}"
OutFile "ArcVault-Setup-${PRODUCT_VERSION}-windows-amd64.exe"
InstallDir "$PROGRAMFILES64\ArcVault"
InstallDirRegKey HKLM "${PRODUCT_UNINST_KEY}" "InstallLocation"
ShowInstDetails show
ShowUnInstDetails show
RequestExecutionLevel admin

; Pages — just welcome, directory, install, finish
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!define MUI_FINISHPAGE_RUN "$INSTDIR\arcvault-setup.exe"
!define MUI_FINISHPAGE_RUN_TEXT "Run ArcVault Setup Wizard"
!define MUI_FINISHPAGE_RUN_NOTCHECKED  ; unchecked by default — user opts in
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "English"

; ============================================================
; Main install section — copy all binaries
; ============================================================
Section "ArcVault" SEC_MAIN
  SectionIn RO
  SetOutPath "$INSTDIR"

  File "coordinator.exe"
  File "agent.exe"
  File "arcvault-setup.exe"

  ; Start Menu shortcuts
  CreateDirectory "$SMPROGRAMS\ArcVault"
  CreateShortCut "$SMPROGRAMS\ArcVault\ArcVault Setup Wizard.lnk" "$INSTDIR\arcvault-setup.exe"
  CreateShortCut "$SMPROGRAMS\ArcVault\Uninstall ArcVault.lnk" "$INSTDIR\uninstall.exe"

  ; Registry + uninstaller
  WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "DisplayName" "${PRODUCT_NAME}"
  WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "UninstallString" "$INSTDIR\uninstall.exe"
  WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "DisplayVersion" "${PRODUCT_VERSION}"
  WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "URLInfoAbout" "${PRODUCT_WEB_SITE}"
  WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "Publisher" "${PRODUCT_PUBLISHER}"
  WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "InstallLocation" "$INSTDIR"
  WriteUninstaller "$INSTDIR\uninstall.exe"
SectionEnd

; ============================================================
; Uninstaller — stop services and remove files
; ============================================================
Section "Uninstall"
  ExecWait '"$INSTDIR\coordinator.exe" uninstall-service'
  ExecWait '"$INSTDIR\agent.exe" uninstall-service'
  Sleep 1000

  RMDir /r "$SMPROGRAMS\ArcVault"
  Delete "$INSTDIR\coordinator.exe"
  Delete "$INSTDIR\agent.exe"
  Delete "$INSTDIR\arcvault-setup.exe"
  Delete "$INSTDIR\config.json"
  Delete "$INSTDIR\agent-config.yaml"
  Delete "$INSTDIR\uninstall.exe"
  RMDir "$INSTDIR"

  DeleteRegKey HKLM "${PRODUCT_UNINST_KEY}"
SectionEnd
