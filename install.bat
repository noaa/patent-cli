@echo off
:: gp-cli Windows installer (CMD / Command Prompt)
:: Usage: curl -fsSL https://raw.githubusercontent.com/noaa/patent-cli/main/install.bat -o "%TEMP%\gp-cli-install.bat" && "%TEMP%\gp-cli-install.bat"
::
:: Requires: Windows 10 build 1803+ (curl and tar are built in)

setlocal EnableDelayedExpansion

set REPO=noaa/patent-cli
set BINARY=gp-cli
set ASSET=gp-cli-windows-amd64.exe
set INST_DIR=%LOCALAPPDATA%\gp-cli

echo.
echo ^>^> Fetching latest release...

:: GitHub API로 최신 태그 조회
curl -fsSL "https://api.github.com/repos/%REPO%/releases/latest" -o "%TEMP%\gp-cli-release.json"
if errorlevel 1 (
    echo ERROR: Failed to contact GitHub API. Check your internet connection.
    exit /b 1
)

:: tag_name 파싱 (PowerShell 없이 순수 CMD로 추출)
for /f "tokens=2 delims=:, " %%A in (
    'findstr /i "tag_name" "%TEMP%\gp-cli-release.json"'
) do (
    set TAG=%%~A
    set TAG=!TAG:"=!
    goto :got_tag
)
:got_tag
del "%TEMP%\gp-cli-release.json" 2>nul

if "!TAG!"=="" (
    echo ERROR: Could not determine latest release tag.
    echo Please visit: https://github.com/%REPO%/releases
    exit /b 1
)

set ZIP_NAME=%ASSET%.zip
set URL=https://github.com/%REPO%/releases/download/!TAG!/%ZIP_NAME%
set TMP_DIR=%TEMP%\gp-cli-install-%RANDOM%

echo ^>^> Installing %BINARY% !TAG! for Windows/amd64
echo.

:: 임시 디렉터리 생성
mkdir "%TMP_DIR%" 2>nul

echo ^>^> Downloading %URL%
curl -fsSL "%URL%" -o "%TMP_DIR%\%ZIP_NAME%"
if errorlevel 1 (
    echo ERROR: Download failed. URL: %URL%
    rmdir /s /q "%TMP_DIR%" 2>nul
    exit /b 1
)

:: ZIP 압축 해제 (tar는 Windows 10 build 1803+ 내장)
echo ^>^> Extracting...
tar -xf "%TMP_DIR%\%ZIP_NAME%" -C "%TMP_DIR%"
if errorlevel 1 (
    echo ERROR: Extraction failed.
    rmdir /s /q "%TMP_DIR%" 2>nul
    exit /b 1
)

:: 설치 디렉터리 생성 및 바이너리 복사
if not exist "%INST_DIR%" mkdir "%INST_DIR%"
copy /y "%TMP_DIR%\%ASSET%" "%INST_DIR%\%BINARY%.exe" >nul
if errorlevel 1 (
    echo ERROR: Could not copy binary to %INST_DIR%
    rmdir /s /q "%TMP_DIR%" 2>nul
    exit /b 1
)

:: 임시 파일 정리
rmdir /s /q "%TMP_DIR%" 2>nul

echo.
echo ^>^> Installed: %INST_DIR%\%BINARY%.exe
echo.

:: 사용자 PATH에 설치 디렉터리 추가 (이미 있으면 건너뜀)
echo !PATH! | findstr /i /c:"%INST_DIR%" >nul 2>&1
if errorlevel 1 (
    :: 레지스트리에서 현재 사용자 PATH 읽기
    for /f "tokens=2*" %%A in (
        'reg query "HKCU\Environment" /v PATH 2^>nul'
    ) do set USER_PATH=%%B

    if "!USER_PATH!"=="" (
        reg add "HKCU\Environment" /v PATH /t REG_EXPAND_SZ /d "%INST_DIR%" /f >nul
    ) else (
        reg add "HKCU\Environment" /v PATH /t REG_EXPAND_SZ /d "!USER_PATH!;%INST_DIR%" /f >nul
    )

    echo ^>^> Added %INST_DIR% to your PATH.
    echo ^>^> Please open a NEW Command Prompt window to use gp-cli.
) else (
    echo ^>^> PATH already contains %INST_DIR%
)

echo.
echo ^>^> Done! Open a new Command Prompt and run: %BINARY% --help
echo.

endlocal
