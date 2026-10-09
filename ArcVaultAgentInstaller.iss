; Inno Setup Script for ArcVault Agent
; Requires Inno Setup 6.0 or later
; Download from: https://jrsoftware.org/isdl.php

#define MyAppName "ArcVault Agent"
#define MyAppVersion "1.0.0"
#define MyAppPublisher "ArcVault"
#define MyAppURL "https://arcvault.com"
#define MyAppExeName "ArcVaultAgent.exe"
#define MyAppServiceName "ArcVaultAgent"
#define NssmExeName "nssm.exe"

[Setup]
; NOTE: The value of AppId uniquely identifies this application.
AppId={{8A9B7C6D-5E4F-3A2B-1C0D-9E8F7A6B5C4D}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppVerName={#MyAppName} {#MyAppVersion}
AppPublisher={#MyAppPublisher}
AppPublisherURL={#MyAppURL}
AppSupportURL={#MyAppURL}
AppUpdatesURL={#MyAppURL}
DefaultDirName={autopf}\{#MyAppName}
DefaultGroupName={#MyAppName}
DisableProgramGroupPage=yes
LicenseFile=
OutputDir=installer_output
OutputBaseFilename=ArcVaultAgent_Setup_{#MyAppVersion}
Compression=lzma2/ultra64
SolidCompression=yes
WizardStyle=modern
PrivilegesRequired=admin
ArchitecturesInstallIn64BitMode=x64
DisableDirPage=no
DisableReadyPage=no
SetupIconFile=
UninstallDisplayIcon={app}\{#MyAppExeName}

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Files]
; Main agent executable
Source: "dist\ArcVaultAgent.exe"; DestDir: "{app}"; Flags: ignoreversion
; NSSM service wrapper
Source: "nssm.exe"; DestDir: "{app}"; Flags: ignoreversion
; Configuration file
Source: "apps\backup-agent\agent\service_config.json"; DestDir: "{app}"; Flags: ignoreversion confirmoverwrite
; Documentation
Source: "apps\backup-agent\agent\SERVICE_TROUBLESHOOTING.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "apps\backup-agent\agent\IMPROVEMENTS_SUMMARY.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "apps\backup-agent\agent\SERVICE_DEPLOYMENT_SUCCESS.md"; DestDir: "{app}"; Flags: ignoreversion
; Diagnostic scripts
Source: "apps\backup-agent\agent\diagnose_service.ps1"; DestDir: "{app}"; Flags: ignoreversion
Source: "apps\backup-agent\agent\reinstall_service.ps1"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
; Start Menu shortcuts
Name: "{group}\Run ArcVault Agent (Console)"; Filename: "{app}\ArcVaultAgent.exe"; Comment: "Run agent in console mode"
Name: "{group}\View Service Output Log"; Filename: "notepad.exe"; Parameters: "{app}\agent_stdout.log"; Comment: "View service output logs"
Name: "{group}\View Service Error Log"; Filename: "notepad.exe"; Parameters: "{app}\agent_stderr.log"; Comment: "View service error logs"
Name: "{group}\Run Diagnostics"; Filename: "powershell.exe"; Parameters: "-ExecutionPolicy Bypass -File ""{app}\diagnose_service.ps1"""; Comment: "Run service diagnostics"; WorkingDir: "{app}"
Name: "{group}\Edit Configuration"; Filename: "notepad.exe"; Parameters: "{app}\service_config.json"; Comment: "Edit service configuration"
Name: "{group}\Troubleshooting Guide"; Filename: "{app}\SERVICE_TROUBLESHOOTING.md"; Comment: "View troubleshooting guide"
Name: "{group}\{cm:UninstallProgram,{#MyAppName}}"; Filename: "{uninstallexe}"

[Run]
; Stop existing service if running
Filename: "{app}\{#NssmExeName}"; Parameters: "stop {#MyAppServiceName}"; Flags: runhidden; StatusMsg: "Stopping existing service..."; Check: ServiceExists
Filename: "{app}\{#NssmExeName}"; Parameters: "remove {#MyAppServiceName} confirm"; Flags: runhidden; StatusMsg: "Removing old service..."; Check: ServiceExists
; Install service with NSSM
Filename: "{app}\{#NssmExeName}"; Parameters: "install {#MyAppServiceName} ""{app}\{#MyAppExeName}"""; Flags: runhidden; StatusMsg: "Installing Windows service..."
; Configure service
Filename: "{app}\{#NssmExeName}"; Parameters: "set {#MyAppServiceName} AppDirectory ""{app}"""; Flags: runhidden
Filename: "{app}\{#NssmExeName}"; Parameters: "set {#MyAppServiceName} DisplayName ""ArcVault Agent Service"""; Flags: runhidden
Filename: "{app}\{#NssmExeName}"; Parameters: "set {#MyAppServiceName} Description ""Runs the ArcVault backup agent"""; Flags: runhidden
Filename: "{app}\{#NssmExeName}"; Parameters: "set {#MyAppServiceName} AppStdout ""{app}\agent_stdout.log"""; Flags: runhidden
Filename: "{app}\{#NssmExeName}"; Parameters: "set {#MyAppServiceName} AppStderr ""{app}\agent_stderr.log"""; Flags: runhidden
Filename: "{app}\{#NssmExeName}"; Parameters: "set {#MyAppServiceName} Start SERVICE_AUTO_START"; Flags: runhidden
; Start the service
Filename: "{app}\{#NssmExeName}"; Parameters: "start {#MyAppServiceName}"; Flags: runhidden; StatusMsg: "Starting ArcVault Agent service..."
; Open documentation
Filename: "{app}\SERVICE_DEPLOYMENT_SUCCESS.md"; Description: "View deployment information"; Flags: postinstall shellexec skipifsilent

[UninstallRun]
; Stop and remove the service with NSSM
Filename: "{app}\{#NssmExeName}"; Parameters: "stop {#MyAppServiceName}"; Flags: runhidden; RunOnceId: "StopService"
Filename: "{app}\{#NssmExeName}"; Parameters: "remove {#MyAppServiceName} confirm"; Flags: runhidden; RunOnceId: "RemoveService"

[Code]
function ServiceExists: Boolean;
var
  ResultCode: Integer;
begin
  // Check if service exists
  Exec('sc.exe', 'query {#MyAppServiceName}', '', SW_HIDE, ewWaitUntilTerminated, ResultCode);
  Result := (ResultCode = 0);
end;

function InitializeSetup(): Boolean;
begin
  Result := True;
  
  // Check if running as administrator
  if not IsAdmin then
  begin
    MsgBox('This installer requires administrative privileges to install a Windows service.' + #13#10 + 
           'Please run the installer as Administrator.', mbError, MB_OK);
    Result := False;
    Exit;
  end;
  
  // Check if port 8443 is available (optional warning)
  // Note: This is a basic check, actual implementation would need more sophisticated detection
end;

procedure CurStepChanged(CurStep: TSetupStep);
var
  ResultCode: Integer;
begin
  if CurStep = ssPostInstall then
  begin
    // Wait a moment for service to start
    Sleep(3000);
    
    // Check if service started successfully
    if Exec('sc.exe', 'query {#MyAppServiceName}', '', SW_HIDE, ewWaitUntilTerminated, ResultCode) then
    begin
      if ResultCode = 0 then
        Log('Service installed and started successfully')
      else
        Log('Warning: Service may not have started correctly');
    end;
  end;
end;

procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
var
  ResultCode: Integer;
begin
  if CurUninstallStep = usPostUninstall then
  begin
    // Clean up log files (optional)
    if MsgBox('Do you want to remove service log files?', mbConfirmation, MB_YESNO) = IDYES then
    begin
      DeleteFile(ExpandConstant('{%TEMP}\arcvault_service.log'));
      DeleteFile('C:\Windows\Temp\arcvault_service.log');
    end;
  end;
end;

[Messages]
WelcomeLabel1=Welcome to the [name] Setup Wizard
WelcomeLabel2=This will install [name/ver] on your computer.%n%nThe ArcVault Agent runs as a Windows service using NSSM to handle backup operations.%n%nIt is recommended that you close all other applications before continuing.
FinishedHeadingLabel=Completing the [name] Setup Wizard
FinishedLabel=The ArcVault Agent service has been installed and started.%n%nThe service is running on port 8443 and configured to start automatically.%n%nView logs in the Start Menu or at: {autopf}\ArcVault Agent\agent_stdout.log

