@echo off
REM Build script for pyenv-win Go implementation

echo Building pyenv-win (Go implementation)...
go build -ldflags="-s -w" -o pyenv.exe ./cmd/pyenv

if %ERRORLEVEL% EQU 0 (
    echo.
    echo Build successful!
    echo Binary: pyenv.exe
    echo.
    echo Run './pyenv.exe --version' to test
) else (
    echo.
    echo Build failed!
    exit /b 1
)
