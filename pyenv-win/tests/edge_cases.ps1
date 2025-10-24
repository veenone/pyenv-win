# Edge Case Testing for pyenv-win Go Implementation
# Tests various error conditions and edge cases

param(
    [string]$Pyenv = "$PSScriptRoot\..\pyenv.exe"
)

Write-Host "============================================" -ForegroundColor Cyan
Write-Host "pyenv-win v4.0.0 Edge Case Testing" -ForegroundColor Cyan
Write-Host "============================================" -ForegroundColor Cyan
Write-Host ""

if (-not (Test-Path $Pyenv)) {
    Write-Host "Error: pyenv binary not found at $Pyenv" -ForegroundColor Red
    exit 1
}

$testsPassed = 0
$testsFailed = 0

function Test-Command {
    param(
        [string]$Name,
        [string]$Command,
        [string]$Args,
        [string]$ExpectedBehavior
    )

    Write-Host "Test: $Name" -ForegroundColor Yellow
    Write-Host "Command: pyenv $Args" -ForegroundColor Gray
    Write-Host "Expected: $ExpectedBehavior" -ForegroundColor Gray

    try {
        if ($Args) {
            $output = & $Command $Args.Split(' ') 2>&1
        } else {
            $output = & $Command 2>&1
        }

        $exitCode = $LASTEXITCODE

        Write-Host "Exit Code: $exitCode" -ForegroundColor Gray
        if ($output) {
            Write-Host "Output: $($output -join ' ')" -ForegroundColor Gray
        }

        # Check if test passed based on exit code and output
        $passed = $false

        if ($ExpectedBehavior -match "error|fail") {
            # Expecting error
            if ($exitCode -ne 0 -or $output -match "error|Error") {
                $passed = $true
            }
        } elseif ($ExpectedBehavior -match "success") {
            # Expecting success
            if ($exitCode -eq 0) {
                $passed = $true
            }
        } else {
            # Just check if it doesn't crash
            $passed = $true
        }

        if ($passed) {
            Write-Host "PASS" -ForegroundColor Green
            $script:testsPassed++
        } else {
            Write-Host "FAIL" -ForegroundColor Red
            $script:testsFailed++
        }
    } catch {
        Write-Host "EXCEPTION: $_" -ForegroundColor Red
        $script:testsFailed++
    }

    Write-Host ""
}

Write-Host "=== Command Line Argument Tests ===" -ForegroundColor Cyan
Write-Host ""

# Test 1: No arguments
Test-Command -Name "No arguments" -Command $Pyenv -Args "" -ExpectedBehavior "Shows usage"

# Test 2: Invalid command
Test-Command -Name "Invalid command" -Command $Pyenv -Args "invalid-command" -ExpectedBehavior "Error message"

# Test 3: Help variations
Test-Command -Name "Help command" -Command $Pyenv -Args "help" -ExpectedBehavior "Success"
Test-Command -Name "Help flag --help" -Command $Pyenv -Args "--help" -ExpectedBehavior "Success"
Test-Command -Name "Help flag -h" -Command $Pyenv -Args "-h" -ExpectedBehavior "Success"

Write-Host ""
Write-Host "=== Install Command Edge Cases ===" -ForegroundColor Cyan
Write-Host ""

# Test 4: Install with no version
Test-Command -Name "Install with no version" -Command $Pyenv -Args "install" -ExpectedBehavior "Shows help or uses current"

# Test 5: Install non-existent version
Test-Command -Name "Install non-existent version" -Command $Pyenv -Args "install 99.99.99" -ExpectedBehavior "Error - version not found"

# Test 6: Install with invalid flags
Test-Command -Name "Install with invalid flag" -Command $Pyenv -Args "install --invalid-flag" -ExpectedBehavior "Ignores or errors"

# Test 7: Multiple conflicting flags
Test-Command -Name "Conflicting flags --32only --64only" -Command $Pyenv -Args "install --32only --64only" -ExpectedBehavior "Error message"

Write-Host ""
Write-Host "=== Version Management Edge Cases ===" -ForegroundColor Cyan
Write-Host ""

# Test 8: Set global to non-existent version
Test-Command -Name "Global with non-existent version" -Command $Pyenv -Args "global 99.99.99" -ExpectedBehavior "Error - not installed"

# Test 9: Local with no arguments
Test-Command -Name "Local with no arguments" -Command $Pyenv -Args "local" -ExpectedBehavior "Shows current local version or error"

# Test 10: Version when no version set
# This might need special setup
Test-Command -Name "Version command" -Command $Pyenv -Args "version" -ExpectedBehavior "Shows version or error"

Write-Host ""
Write-Host "=== Uninstall Edge Cases ===" -ForegroundColor Cyan
Write-Host ""

# Test 11: Uninstall non-existent version
Test-Command -Name "Uninstall non-existent" -Command $Pyenv -Args "uninstall 99.99.99" -ExpectedBehavior "Error - not installed"

# Test 12: Uninstall with no version
Test-Command -Name "Uninstall with no version" -Command $Pyenv -Args "uninstall" -ExpectedBehavior "Shows usage"

Write-Host ""
Write-Host "=== Latest Command Edge Cases ===" -ForegroundColor Cyan
Write-Host ""

# Test 13: Latest with no prefix
Test-Command -Name "Latest with no prefix" -Command $Pyenv -Args "latest" -ExpectedBehavior "Shows usage"

# Test 14: Latest with invalid prefix
Test-Command -Name "Latest with invalid prefix" -Command $Pyenv -Args "latest 99.99" -ExpectedBehavior "Error - no matching version"

Write-Host ""
Write-Host "=== List Commands ===" -ForegroundColor Cyan
Write-Host ""

# Test 15: List commands
Test-Command -Name "List all commands" -Command $Pyenv -Args "commands" -ExpectedBehavior "Success"

# Test 16: List versions
Test-Command -Name "List installed versions" -Command $Pyenv -Args "versions" -ExpectedBehavior "Success or shows none installed"

# Test 17: List available versions
Test-Command -Name "List available versions" -Command $Pyenv -Args "install --list" -ExpectedBehavior "Success"

Write-Host ""
Write-Host "=== Special Characters and Spaces ===" -ForegroundColor Cyan
Write-Host ""

# Test 18: Command with trailing spaces
Test-Command -Name "Command with trailing space" -Command $Pyenv -Args "versions " -ExpectedBehavior "Success"

# Test 19: Version flag variations
Test-Command -Name "Version flag --version" -Command $Pyenv -Args "--version" -ExpectedBehavior "Success"
Test-Command -Name "Version flag -v" -Command $Pyenv -Args "-v" -ExpectedBehavior "Success or error"

Write-Host ""
Write-Host "=== Error Handling ===" -ForegroundColor Cyan
Write-Host ""

# Test 20: Very long command
Test-Command -Name "Very long argument" -Command $Pyenv -Args ("install " + ("a" * 1000)) -ExpectedBehavior "Error or handles gracefully"

# Summary
Write-Host ""
Write-Host "============================================" -ForegroundColor Cyan
Write-Host "Test Summary" -ForegroundColor Cyan
Write-Host "============================================" -ForegroundColor Cyan
Write-Host "Tests Passed: $testsPassed" -ForegroundColor Green
Write-Host "Tests Failed: $testsFailed" -ForegroundColor Red
Write-Host "Total Tests: $($testsPassed + $testsFailed)" -ForegroundColor Yellow
Write-Host ""

if ($testsFailed -eq 0) {
    Write-Host "All tests passed!" -ForegroundColor Green
    exit 0
} else {
    Write-Host "Some tests failed. Please review the output above." -ForegroundColor Red
    exit 1
}
