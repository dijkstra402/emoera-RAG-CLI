Unicode True

!include "MUI2.nsh"
!include "LogicLib.nsh"

!ifndef VERSION
  !error "VERSION is required"
!endif
!ifndef BINARY_PATH
  !error "BINARY_PATH is required"
!endif
!ifndef OUTPUT_FILE
  !error "OUTPUT_FILE is required"
!endif

Name "Emoera Agent CLI"
OutFile "${OUTPUT_FILE}"
InstallDir "$LOCALAPPDATA\Programs\Emoera CLI"
InstallDirRegKey HKCU "Software\Emoera CLI" "InstallDir"
RequestExecutionLevel user
SetCompressor /SOLID lzma

!define MUI_ABORTWARNING
!define MUI_ICON "${NSISDIR}\Contrib\Graphics\Icons\modern-install.ico"
!define MUI_UNICON "${NSISDIR}\Contrib\Graphics\Icons\modern-uninstall.ico"

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "SimpChinese"
!insertmacro MUI_LANGUAGE "English"

Section "Emoera Agent CLI" SecMain
  SetOutPath "$INSTDIR"
  File /oname=emoera.exe "${BINARY_PATH}"
  WriteUninstaller "$INSTDIR\uninstall.exe"
  WriteRegStr HKCU "Software\Emoera CLI" "InstallDir" "$INSTDIR"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Emoera CLI" "DisplayName" "Emoera Agent CLI"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Emoera CLI" "DisplayVersion" "${VERSION}"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Emoera CLI" "Publisher" "Emoera"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Emoera CLI" "URLInfoAbout" "https://github.com/dijkstra402/emoera-RAG-CLI"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Emoera CLI" "UninstallString" '"$INSTDIR\uninstall.exe"'
  WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Emoera CLI" "NoModify" 1
  WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Emoera CLI" "NoRepair" 1

  ReadRegStr $0 HKCU "Environment" "Path"
  StrCpy $1 ";$INSTDIR"
  StrLen $2 $1
  StrCpy $3 $0 $2 -$2
  ${If} $3 != $1
    ${If} $0 == ""
      WriteRegExpandStr HKCU "Environment" "Path" "$INSTDIR"
    ${Else}
      WriteRegExpandStr HKCU "Environment" "Path" "$0;$INSTDIR"
    ${EndIf}
  ${EndIf}
  SendMessage ${HWND_BROADCAST} ${WM_SETTINGCHANGE} 0 "STR:Environment" /TIMEOUT=5000
SectionEnd

Section "Uninstall"
  Delete "$INSTDIR\emoera.exe"
  Delete "$INSTDIR\uninstall.exe"
  RMDir "$INSTDIR"
  DeleteRegKey HKCU "Software\Emoera CLI"
  DeleteRegKey HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Emoera CLI"

  ReadRegStr $0 HKCU "Environment" "Path"
  StrCpy $1 ";$INSTDIR"
  StrLen $2 $1
  StrCpy $3 $0 $2 -$2
  ${If} $3 == $1
    StrCpy $0 $0 -$2
    WriteRegExpandStr HKCU "Environment" "Path" "$0"
  ${EndIf}
  SendMessage ${HWND_BROADCAST} ${WM_SETTINGCHANGE} 0 "STR:Environment" /TIMEOUT=5000
SectionEnd
