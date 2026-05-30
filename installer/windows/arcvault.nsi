; ArcVault Installer (NSIS Modern UI 2)
; Supports both Coordinator and Agent installation with guided setup

!include "MUI2.nsh"
!include "x64.nsh"
!include "nsDialogs.nsh"
!include "LogicLib.nsh"
!include "Sections.nsh"

; Product info
!define PRODUCT_NAME "ArcVault"
!define PRODUCT_VERSION "0.2.1"
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

; ============================================================
; Variables
; ============================================================
Var CoordinatorPort
Var CoordinatorURL
Var AgentID
Var AgentAuthToken

; Dialog handles
Var Dialog
Var Ctrl_CoordPort
Var Ctrl_CoordURL
Var Ctrl_AgentID
Var Ctrl_AgentToken
Var Ctrl_AgentTokenLabel

; ============================================================
; Page definitions
; ============================================================
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_COMPONENTS
!insertmacro MUI_PAGE_DIRECTORY
Page custom CoordConfigPageCreate CoordConfigPageLeave
Page custom AgentConfigPageCreate AgentConfigPageLeave
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "English"

; ============================================================
; Page 1: Coordinator config (skipped if coordinator not selected)
; ============================================================
Function CoordConfigPageCreate
  ; Skip if coordinator section not selected (section 0)
  Push $0
  SectionGetFlags 0 $0
  IntOp $0 $0 & ${SF_SELECTED}
  Pop $0
  IntCmp $0 ${SF_SELECTED} +2
    Abort ; skip page

  !insertmacro MUI_HEADER_TEXT "Coordinator Settings" "Configure the ArcVault Coordinator."

  nsDialogs::Create 1018
  Pop $Dialog
  ${If} $Dialog == error
    Abort
  ${EndIf}

  ${NSD_CreateLabel} 0 0 100% 12u "Dashboard port (default: 8080):"
  Pop $0
  ${NSD_CreateNumber} 0 14u 60u 14u "8080"
  Pop $Ctrl_CoordPort

  ${NSD_CreateLabel} 0 36u 100% 24u "The coordinator dashboard will be accessible at http://localhost:<port> after installation."
  Pop $0

  nsDialogs::Show
FunctionEnd

Function CoordConfigPageLeave
  ${NSD_GetText} $Ctrl_CoordPort $CoordinatorPort
  ${If} $CoordinatorPort == ""
    StrCpy $CoordinatorPort "8080"
  ${EndIf}
FunctionEnd

; ============================================================
; Page 2: Agent config (skipped if agent not selected)
; ============================================================
Function AgentConfigPageCreate
  ; Skip if agent section not selected (section 1)
  Push $0
  SectionGetFlags 1 $0
  IntOp $0 $0 & ${SF_SELECTED}
  Pop $0
  IntCmp $0 ${SF_SELECTED} +2
    Abort ; skip page

  !insertmacro MUI_HEADER_TEXT "Agent Settings" "Configure the ArcVault Agent."

  nsDialogs::Create 1018
  Pop $Dialog
  ${If} $Dialog == error
    Abort
  ${EndIf}

  ${NSD_CreateLabel} 0 0 100% 12u "Agent ID (unique name for this machine, e.g. agent-pc1):"
  Pop $0
  ${NSD_CreateText} 0 14u 100% 14u "agent-01"
  Pop $Ctrl_AgentID

  ${NSD_CreateLabel} 0 36u 100% 12u "Coordinator URL (e.g. http://192.168.1.10:8080):"
  Pop $0
  ${NSD_CreateText} 0 50u 100% 14u "http://localhost:8080"
  Pop $Ctrl_CoordURL

  ; Check if coordinator is also being installed on this machine
  Push $0
  SectionGetFlags 0 $0
  IntOp $0 $0 & ${SF_SELECTED}
  Pop $0
  ${If} $0 == ${SF_SELECTED}
    ; Coordinator is being installed here — token will be auto-generated
    ${NSD_CreateLabel} 0 72u 100% 24u "Auth token will be auto-generated after the coordinator starts.$\nA copy will be saved to your Desktop as agent-token.txt."
    Pop $Ctrl_AgentTokenLabel
    ; Hidden text control to hold auto-gen placeholder
    ${NSD_CreateText} 0 100u 100% 14u "(auto-generated)"
    Pop $Ctrl_AgentToken
    EnableWindow $Ctrl_AgentToken 0
  ${Else}
    ; Agent-only install — user must supply token from existing coordinator
    ${NSD_CreateLabel} 0 72u 100% 12u "Auth token (from coordinator CLI: coordinator create-agent-token <id>):"
    Pop $Ctrl_AgentTokenLabel
    ${NSD_CreateText} 0 86u 100% 14u ""
    Pop $Ctrl_AgentToken
  ${EndIf}

  nsDialogs::Show
FunctionEnd

Function AgentConfigPageLeave
  ${NSD_GetText} $Ctrl_AgentID $AgentID
  ${NSD_GetText} $Ctrl_CoordURL $CoordinatorURL

  ${If} $AgentID == ""
    StrCpy $AgentID "agent-01"
  ${EndIf}
  ${If} $CoordinatorURL == ""
    StrCpy $CoordinatorURL "http://localhost:8080"
  ${EndIf}

  ; Only read token field if agent-only install (coordinator not selected)
  Push $0
  SectionGetFlags 0 $0
  IntOp $0 $0 & ${SF_SELECTED}
  Pop $0
  ${If} $0 != ${SF_SELECTED}
    ${NSD_GetText} $Ctrl_AgentToken $AgentAuthToken
  ${EndIf}
FunctionEnd

; ============================================================
; Installation sections
; ============================================================
Section "Coordinator"
  SectionIn RO
  SetOutPath "$INSTDIR"

  ; Copy coordinator binary and setup wizard
  File "coordinator.exe"
  File "arcvault-setup.exe"

  ; Write config.json — coordinator reads this on startup
  FileOpen $0 "$INSTDIR\config.json" w
  FileWrite $0 '{$\n'
  FileWrite $0 '  "port": $CoordinatorPort,$\n'
  FileWrite $0 '  "db_path": "$INSTDIR\\arcvault.db",$\n'
  FileWrite $0 '  "log_level": "info"$\n'
  FileWrite $0 '}$\n'
  FileClose $0

  ; Shortcuts
  CreateDirectory "$SMPROGRAMS\ArcVault"
  CreateShortCut "$SMPROGRAMS\ArcVault\ArcVault Setup.lnk" "$INSTDIR\arcvault-setup.exe"
  CreateShortCut "$SMPROGRAMS\ArcVault\Uninstall ArcVault.lnk" "$INSTDIR\uninstall.exe"

  ; Remove old service if present, then install fresh
  ExecWait '"$INSTDIR\coordinator.exe" uninstall-service'
  Sleep 1000
  ExecWait '"$INSTDIR\coordinator.exe" install-service'
  ExecWait 'sc config ArcVaultCoordinator start= auto'
  ExecWait 'sc start ArcVaultCoordinator'

  ; Give coordinator a moment to initialise its DB before token generation
  Sleep 3000

  ; If agent is also being installed, generate a token now while coordinator is running
  Push $0
  SectionGetFlags 1 $0
  IntOp $0 $0 & ${SF_SELECTED}
  Pop $0
  ${If} $0 == ${SF_SELECTED}
    ; Generate token — coordinator writes only the token to stdout with --token-only flag
    nsExec::ExecToStack '"$INSTDIR\coordinator.exe" create-agent-token "$AgentID" --token-only'
    Pop $0          ; exit code
    Pop $AgentAuthToken  ; token string (stdout)

    ; Strip any trailing whitespace/newlines
    ; Write token to Desktop for the user's reference
    FileOpen $0 "$DESKTOP\agent-token.txt" w
    FileWrite $0 "ArcVault Agent Token$\r$\n"
    FileWrite $0 "===================$\r$\n"
    FileWrite $0 "Agent ID : $AgentID$\r$\n"
    FileWrite $0 "Token    : $AgentAuthToken$\r$\n"
    FileWrite $0 "$\r$\nKeep this file safe. You will need this token to register additional agents.$\r$\n"
    FileClose $0

    DetailPrint "Agent token generated and saved to Desktop\agent-token.txt"
  ${EndIf}
SectionEnd

Section "Agent"
  SetOutPath "$INSTDIR"

  ; Copy agent binary and setup wizard
  File "agent.exe"
  File "arcvault-setup.exe"

  ; Shortcuts
  CreateDirectory "$SMPROGRAMS\ArcVault"
  CreateShortCut "$SMPROGRAMS\ArcVault\ArcVault Setup.lnk" "$INSTDIR\arcvault-setup.exe"
  CreateShortCut "$SMPROGRAMS\ArcVault\Uninstall ArcVault.lnk" "$INSTDIR\uninstall.exe"

  ; Write agent-config.yaml — all three required fields must be present
  FileOpen $0 "$INSTDIR\agent-config.yaml" w
  FileWrite $0 "agent_id: $AgentID$\n"
  FileWrite $0 "coordinator_url: $CoordinatorURL$\n"
  FileWrite $0 "auth_token: $AgentAuthToken$\n"
  FileWrite $0 "version: ${PRODUCT_VERSION}$\n"
  FileClose $0

  ; Remove old service if present, then install fresh
  ExecWait '"$INSTDIR\agent.exe" uninstall-service'
  Sleep 1000
  ExecWait '"$INSTDIR\agent.exe" install-service'
  ExecWait 'sc config ArcVaultAgent start= auto'
  ExecWait 'sc start ArcVaultAgent'
SectionEnd

; ============================================================
; Post section — registry + uninstaller
; ============================================================
Section -Post
  WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "DisplayName" "${PRODUCT_NAME}"
  WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "UninstallString" "$INSTDIR\uninstall.exe"
  WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "DisplayVersion" "${PRODUCT_VERSION}"
  WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "URLInfoAbout" "${PRODUCT_WEB_SITE}"
  WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "Publisher" "${PRODUCT_PUBLISHER}"
  WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "InstallLocation" "$INSTDIR"
  WriteUninstaller "$INSTDIR\uninstall.exe"
SectionEnd

; ============================================================
; Uninstaller
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

; ============================================================
; Section descriptions (shown in component selection page)
; ============================================================
LangString DESC_SEC_COORDINATOR ${LANG_ENGLISH} "Install ArcVault Coordinator — the central server and web dashboard."
LangString DESC_SEC_AGENT ${LANG_ENGLISH} "Install ArcVault Agent — runs on machines that perform backups."

!insertmacro MUI_FUNCTION_DESCRIPTION_BEGIN
  !insertmacro MUI_DESCRIPTION_TEXT ${SEC_COORDINATOR} $(DESC_SEC_COORDINATOR)
  !insertmacro MUI_DESCRIPTION_TEXT ${SEC_AGENT} $(DESC_SEC_AGENT)
!insertmacro MUI_FUNCTION_DESCRIPTION_END
