; ArcVault Installer (NSIS Modern UI 2)
; Supports both Coordinator and Agent installation with guided setup

!include "MUI2.nsh"
!include "x64.nsh"
!include "nsDialogs.nsh"
!include "LogicLib.nsh"

; Product info
!define PRODUCT_NAME "ArcVault"
!define PRODUCT_VERSION "1.1.0"
!define PRODUCT_PUBLISHER "ArcVault Team"
!define PRODUCT_WEB_SITE "https://github.com/castrokren/ArcVault"
!define PRODUCT_UNINST_KEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_NAME}"

; Default installation folder
InstallDir "$PROGRAMFILES\ArcVault"

; Installer attributes
Name "${PRODUCT_NAME} ${PRODUCT_VERSION}"
OutFile "ArcVault-Setup-${PRODUCT_VERSION}-windows-amd64.exe"
InstallDirRegKey HKLM "${PRODUCT_UNINST_KEY}" "InstallLocation"
ShowInstDetails show
ShowUnInstDetails show
RequestExecutionLevel admin

; Variables
Var ComponentSelection
Var ComponentCoordinator
Var ComponentAgent

; Modern UI 2 Configuration
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_COMPONENTS
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_LANGUAGE "English"

; Installation sections
Section "Coordinator" SEC_COORDINATOR
  SectionIn RO
  SetOutPath "$INSTDIR"

  ; Copy coordinator binary
  File "coordinator.exe"
  File "arcvault-setup.exe"

  ; Create shortcuts
  CreateDirectory "$SMPROGRAMS\ArcVault"
  CreateShortCut "$SMPROGRAMS\ArcVault\ArcVault Setup.lnk" "$INSTDIR\arcvault-setup.exe"
  CreateShortCut "$SMPROGRAMS\ArcVault\Uninstall ArcVault.lnk" "$UNINSTDIR\uninstall.exe"

  ; Run setup wizard
  ExecWait '"$INSTDIR\arcvault-setup.exe"'

  ; Install service
  ExecWait '"$INSTDIR\coordinator.exe" install-service'
SectionEnd

Section "Agent" SEC_AGENT
  SetOutPath "$INSTDIR"

  ; Copy agent binary
  File "agent.exe"
  File "arcvault-setup.exe"

  ; Create shortcuts
  CreateDirectory "$SMPROGRAMS\ArcVault"
  CreateShortCut "$SMPROGRAMS\ArcVault\ArcVault Setup.lnk" "$INSTDIR\arcvault-setup.exe"
  CreateShortCut "$SMPROGRAMS\ArcVault\Uninstall ArcVault.lnk" "$UNINSTDIR\uninstall.exe"

  ; Run setup wizard
  ExecWait '"$INSTDIR\arcvault-setup.exe"'

  ; Install service
  ExecWait '"$INSTDIR\agent.exe" install-service'
SectionEnd

Section -Post
  WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "DisplayName" "${PRODUCT_NAME}"
  WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "UninstallString" "$UNINSTDIR\uninstall.exe"
  WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "DisplayVersion" "${PRODUCT_VERSION}"
  WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "URLInfoAbout" "${PRODUCT_WEB_SITE}"
  WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "Publisher" "${PRODUCT_PUBLISHER}"
  WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "InstallLocation" "$INSTDIR"
  WriteUninstaller "$UNINSTDIR\uninstall.exe"
SectionEnd

; Uninstaller
Section "Uninstall"
  ; Stop services
  ExecWait '"$INSTDIR\coordinator.exe" uninstall-service' 0
  ExecWait '"$INSTDIR\agent.exe" uninstall-service' 0

  Sleep 1000

  ; Remove files
  RMDir /r "$SMPROGRAMS\ArcVault"
  Delete "$INSTDIR\coordinator.exe"
  Delete "$INSTDIR\agent.exe"
  Delete "$INSTDIR\arcvault-setup.exe"
  Delete "$INSTDIR\uninstall.exe"
  RMDir "$INSTDIR"

  DeleteRegKey HKLM "${PRODUCT_UNINST_KEY}"
SectionEnd

; Section descriptions
LangString DESC_SEC_COORDINATOR ${LANG_ENGLISH} "Install ArcVault Coordinator (server)"
LangString DESC_SEC_AGENT ${LANG_ENGLISH} "Install ArcVault Agent (client)"

!insertmacro MUI_FUNCTION_DESCRIPTION_BEGIN
  !insertmacro MUI_DESCRIPTION_TEXT ${SEC_COORDINATOR} $(DESC_SEC_COORDINATOR)
  !insertmacro MUI_DESCRIPTION_TEXT ${SEC_AGENT} $(DESC_SEC_AGENT)
!insertmacro MUI_FUNCTION_DESCRIPTION_END
