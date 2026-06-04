@echo off
:: gp-cli Windows uninstaller (CMD)
:: Usage: curl -fsSL https://raw.githubusercontent.com/noaa/patent-cli/main/uninstall.bat -o "%TEMP%\gp-cli-uninstall.bat" && "%TEMP%\gp-cli-uninstall.bat"

setlocal EnableDelayedExpansion

set BINARY=gp-cli
set INST_DIR=%LOCALAPPDATA%\gp-cli
set CONFIG_DIR=%APPDATA%\patent-cli
set BINARY_PATH=%INST_DIR%\%BINARY%.exe
set REMOVED=0

echo.

if exist "%BINARY_PATH%" (
    del /f /q "%BINARY_PATH%"
    echo ^>^> Removed: %BINARY_PATH%
    set REMOVED=1
) else (
    echo ^>^> Binary not found: %BINARY_PATH%
)

if exist "%INST_DIR%" (
    rmdir /s /q "%INST_DIR%" 2>nul
    echo ^>^> Removed dir: %INST_DIR%
    set REMOVED=1
)

if exist "%CONFIG_DIR%" (
    rmdir /s /q "%CONFIG_DIR%"
    echo ^>^> Removed config: %CONFIG_DIR%
    set REMOVED=1
) else (
    echo ^>^> Config dir not found: %CONFIG_DIR%
)

:: Remove from user PATH via registry
for /f "tokens=2*" %%A in (
    'reg query "HKCU\Environment" /v PATH 2^>nul'
) do set USER_PATH=%%B

echo !USER_PATH! | findstr /i /c:"%INST_DIR%" >nul 2>&1
if not errorlevel 1 (
    :: INST_DIR를 PATH에서 제거
    set NEW_PATH=
    for %%P in ("!USER_PATH:;=" "!") do (
        if /i not "%%~P"=="%INST_DIR%" (
            if defined NEW_PATH (
                set NEW_PATH=!NEW_PATH!;%%~P
            ) else (
                set NEW_PATH=%%~P
            )
        )
    )
    reg add "HKCU\Environment" /v PATH /t REG_EXPAND_SZ /d "!NEW_PATH!" /f >nul
    echo ^>^> Removed %INST_DIR% from PATH.
    echo ^>^> Please open a NEW Command Prompt for PATH changes to take effect.
    set REMOVED=1
)

echo.
if "!REMOVED!"=="1" (
    echo ^>^> gp-cli uninstalled.
) else (
    echo ^>^> Nothing to remove.
)
echo.

endlocal
