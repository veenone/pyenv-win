@echo off
REM Performance Benchmark Script for pyenv-win Go vs VBScript
REM Compares execution times for various commands

setlocal enabledelayedexpansion

echo ============================================
echo pyenv-win Performance Benchmark
echo Go v4.0.0 vs VBScript v3.1.1
echo ============================================
echo.

REM Set paths
set GO_PYENV=%~dp0..\pyenv.exe
set VBS_PYENV=C:\Users\ARaha\.pyenv\pyenv-win\bin\pyenv

REM Check if both implementations exist
if not exist "%GO_PYENV%" (
    echo Error: Go implementation not found at %GO_PYENV%
    exit /b 1
)

if not exist "%VBS_PYENV%" (
    echo Error: VBScript implementation not found at %VBS_PYENV%
    exit /b 1
)

echo Test 1: Version Display
echo ------------------------
echo.

echo [VBScript] pyenv --version
powershell -Command "Measure-Command { & '%VBS_PYENV%' --version | Out-Null } | Select-Object -ExpandProperty TotalMilliseconds"
set VBS_VERSION_TIME=!errorlevel!

echo.
echo [Go] pyenv --version
powershell -Command "Measure-Command { & '%GO_PYENV%' --version | Out-Null } | Select-Object -ExpandProperty TotalMilliseconds"
set GO_VERSION_TIME=!errorlevel!

echo.
echo.

echo Test 2: List Commands
echo ---------------------
echo.

echo [VBScript] pyenv commands
powershell -Command "Measure-Command { & '%VBS_PYENV%' commands | Out-Null } | Select-Object -ExpandProperty TotalMilliseconds"

echo.
echo [Go] pyenv commands
powershell -Command "Measure-Command { & '%GO_PYENV%' commands | Out-Null } | Select-Object -ExpandProperty TotalMilliseconds"

echo.
echo.

echo Test 3: List Installed Versions
echo --------------------------------
echo.

echo [VBScript] pyenv versions
powershell -Command "Measure-Command { & '%VBS_PYENV%' versions | Out-Null } | Select-Object -ExpandProperty TotalMilliseconds"

echo.
echo [Go] pyenv versions
powershell -Command "Measure-Command { & '%GO_PYENV%' versions | Out-Null } | Select-Object -ExpandProperty TotalMilliseconds"

echo.
echo.

echo Test 4: Show Help
echo -----------------
echo.

echo [VBScript] pyenv install --help
powershell -Command "Measure-Command { & '%VBS_PYENV%' install --help | Out-Null } | Select-Object -ExpandProperty TotalMilliseconds"

echo.
echo [Go] pyenv install --help
powershell -Command "Measure-Command { & '%GO_PYENV%' install --help | Out-Null } | Select-Object -ExpandProperty TotalMilliseconds"

echo.
echo.

echo Test 5: List Available Versions (Heavy Operation)
echo --------------------------------------------------
echo.

echo [VBScript] pyenv install --list
powershell -Command "$time = Measure-Command { & '%VBS_PYENV%' install --list | Out-Null }; Write-Output $time.TotalMilliseconds"

echo.
echo [Go] pyenv install --list
powershell -Command "$time = Measure-Command { & '%GO_PYENV%' install --list | Out-Null }; Write-Output $time.TotalMilliseconds"

echo.
echo.

echo ============================================
echo Benchmark Complete
echo ============================================
echo.
echo Note: Lower times are better
echo Times are in milliseconds

pause
