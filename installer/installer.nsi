!define APP_NAME "LocalAI Hub"
!define APP_EXE "LocalAI.exe"
!define REG_KEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_NAME}"

OutFile "..\dist\LocalAI_Hub_Setup.exe"
InstallDir "$LOCALAPPDATA\${APP_NAME}"
RequestExecutionLevel user

!include "MUI2.nsh"
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_LANGUAGE "English"

Section "Install"
  SetOutPath "$INSTDIR"
  File "..\dist\${APP_EXE}"

  CreateDirectory "$SMPROGRAMS\${APP_NAME}"
  CreateShortCut "$SMPROGRAMS\${APP_NAME}\${APP_NAME}.lnk" "$INSTDIR\${APP_EXE}"
  CreateShortCut "$DESKTOP\${APP_NAME}.lnk" "$INSTDIR\${APP_EXE}"

  WriteUninstaller "$INSTDIR\uninstall.exe"

  WriteRegStr HKCU "${REG_KEY}" "DisplayName" "${APP_NAME}"
  WriteRegStr HKCU "${REG_KEY}" "UninstallString" "$INSTDIR\uninstall.exe"
  WriteRegStr HKCU "${REG_KEY}" "DisplayIcon" "$INSTDIR\${APP_EXE}"
  WriteRegStr HKCU "${REG_KEY}" "DisplayVersion" "1.0.0"
  WriteRegStr HKCU "${REG_KEY}" "Publisher" "LocalAI Hub"
SectionEnd

Section "Uninstall"
  Delete "$INSTDIR\${APP_EXE}"
  Delete "$INSTDIR\uninstall.exe"
  RMDir "$INSTDIR"

  Delete "$SMPROGRAMS\${APP_NAME}\${APP_NAME}.lnk"
  RMDir "$SMPROGRAMS\${APP_NAME}"
  Delete "$DESKTOP\${APP_NAME}.lnk"

  DeleteRegKey HKCU "${REG_KEY}"
SectionEnd
