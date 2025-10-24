# Performance Benchmark Script for pyenv-win Go vs VBScript
# PowerShell version for accurate timing

param(
    [string]$GoPyenv = "$PSScriptRoot\..\pyenv.exe",
    [string]$VbsPyenv = "$env:USERPROFILE\.pyenv\pyenv-win\bin\pyenv",
    [int]$Iterations = 3
)

Write-Host "============================================" -ForegroundColor Cyan
Write-Host "pyenv-win Performance Benchmark" -ForegroundColor Cyan
Write-Host "Go v4.0.0 vs VBScript v3.1.1" -ForegroundColor Cyan
Write-Host "============================================" -ForegroundColor Cyan
Write-Host ""

# Check if both implementations exist
if (-not (Test-Path $GoPyenv)) {
    Write-Host "Error: Go implementation not found at $GoPyenv" -ForegroundColor Red
    exit 1
}

if (-not (Test-Path $VbsPyenv)) {
    Write-Host "Error: VBScript implementation not found at $VbsPyenv" -ForegroundColor Red
    exit 1
}

Write-Host "Running $Iterations iterations per test..." -ForegroundColor Yellow
Write-Host ""

# Function to measure command execution
function Measure-PyenvCommand {
    param(
        [string]$Implementation,
        [string]$Command,
        [string]$Args
    )

    $times = @()
    for ($i = 0; $i -lt $Iterations; $i++) {
        $time = Measure-Command {
            if ($Args) {
                & $Command $Args.Split(' ') | Out-Null
            } else {
                & $Command | Out-Null
            }
        }
        $times += $time.TotalMilliseconds
    }

    $avgTime = ($times | Measure-Object -Average).Average
    $minTime = ($times | Measure-Object -Minimum).Minimum
    $maxTime = ($times | Measure-Object -Maximum).Maximum

    return @{
        Average = [math]::Round($avgTime, 2)
        Min = [math]::Round($minTime, 2)
        Max = [math]::Round($maxTime, 2)
    }
}

# Results array
$results = @()

# Test 1: Version Display
Write-Host "Test 1: Version Display (pyenv --version)" -ForegroundColor Green
Write-Host "----------------------------------------" -ForegroundColor Green
$vbsResult = Measure-PyenvCommand -Implementation "VBScript" -Command $VbsPyenv -Args "--version"
Write-Host "[VBScript] Avg: $($vbsResult.Average)ms, Min: $($vbsResult.Min)ms, Max: $($vbsResult.Max)ms"

$goResult = Measure-PyenvCommand -Implementation "Go" -Command $GoPyenv -Args "--version"
Write-Host "[Go]       Avg: $($goResult.Average)ms, Min: $($goResult.Min)ms, Max: $($goResult.Max)ms" -ForegroundColor Cyan

$speedup = [math]::Round($vbsResult.Average / $goResult.Average, 2)
Write-Host "Speedup: ${speedup}x faster" -ForegroundColor Yellow
Write-Host ""

$results += [PSCustomObject]@{
    Test = "Version Display"
    VBScript_Avg = $vbsResult.Average
    Go_Avg = $goResult.Average
    Speedup = "${speedup}x"
}

# Test 2: List Commands
Write-Host "Test 2: List Commands (pyenv commands)" -ForegroundColor Green
Write-Host "--------------------------------------" -ForegroundColor Green
$vbsResult = Measure-PyenvCommand -Implementation "VBScript" -Command $VbsPyenv -Args "commands"
Write-Host "[VBScript] Avg: $($vbsResult.Average)ms, Min: $($vbsResult.Min)ms, Max: $($vbsResult.Max)ms"

$goResult = Measure-PyenvCommand -Implementation "Go" -Command $GoPyenv -Args "commands"
Write-Host "[Go]       Avg: $($goResult.Average)ms, Min: $($goResult.Min)ms, Max: $($goResult.Max)ms" -ForegroundColor Cyan

$speedup = [math]::Round($vbsResult.Average / $goResult.Average, 2)
Write-Host "Speedup: ${speedup}x faster" -ForegroundColor Yellow
Write-Host ""

$results += [PSCustomObject]@{
    Test = "List Commands"
    VBScript_Avg = $vbsResult.Average
    Go_Avg = $goResult.Average
    Speedup = "${speedup}x"
}

# Test 3: List Installed Versions
Write-Host "Test 3: List Installed Versions (pyenv versions)" -ForegroundColor Green
Write-Host "------------------------------------------------" -ForegroundColor Green
$vbsResult = Measure-PyenvCommand -Implementation "VBScript" -Command $VbsPyenv -Args "versions"
Write-Host "[VBScript] Avg: $($vbsResult.Average)ms, Min: $($vbsResult.Min)ms, Max: $($vbsResult.Max)ms"

$goResult = Measure-PyenvCommand -Implementation "Go" -Command $GoPyenv -Args "versions"
Write-Host "[Go]       Avg: $($goResult.Average)ms, Min: $($goResult.Min)ms, Max: $($goResult.Max)ms" -ForegroundColor Cyan

$speedup = [math]::Round($vbsResult.Average / $goResult.Average, 2)
Write-Host "Speedup: ${speedup}x faster" -ForegroundColor Yellow
Write-Host ""

$results += [PSCustomObject]@{
    Test = "List Versions"
    VBScript_Avg = $vbsResult.Average
    Go_Avg = $goResult.Average
    Speedup = "${speedup}x"
}

# Test 4: Show Help
Write-Host "Test 4: Show Help (pyenv install --help)" -ForegroundColor Green
Write-Host "----------------------------------------" -ForegroundColor Green
$vbsResult = Measure-PyenvCommand -Implementation "VBScript" -Command $VbsPyenv -Args "install --help"
Write-Host "[VBScript] Avg: $($vbsResult.Average)ms, Min: $($vbsResult.Min)ms, Max: $($vbsResult.Max)ms"

$goResult = Measure-PyenvCommand -Implementation "Go" -Command $GoPyenv -Args "install --help"
Write-Host "[Go]       Avg: $($goResult.Average)ms, Min: $($goResult.Min)ms, Max: $($goResult.Max)ms" -ForegroundColor Cyan

$speedup = [math]::Round($vbsResult.Average / $goResult.Average, 2)
Write-Host "Speedup: ${speedup}x faster" -ForegroundColor Yellow
Write-Host ""

$results += [PSCustomObject]@{
    Test = "Show Help"
    VBScript_Avg = $vbsResult.Average
    Go_Avg = $goResult.Average
    Speedup = "${speedup}x"
}

# Test 5: List Available Versions (Heavy Operation)
Write-Host "Test 5: List Available Versions (pyenv install --list)" -ForegroundColor Green
Write-Host "------------------------------------------------------" -ForegroundColor Green
Write-Host "Note: This is a heavy operation that reads the version cache" -ForegroundColor Yellow

$vbsResult = Measure-PyenvCommand -Implementation "VBScript" -Command $VbsPyenv -Args "install --list"
Write-Host "[VBScript] Avg: $($vbsResult.Average)ms, Min: $($vbsResult.Min)ms, Max: $($vbsResult.Max)ms"

$goResult = Measure-PyenvCommand -Implementation "Go" -Command $GoPyenv -Args "install --list"
Write-Host "[Go]       Avg: $($goResult.Average)ms, Min: $($goResult.Min)ms, Max: $($goResult.Max)ms" -ForegroundColor Cyan

$speedup = [math]::Round($vbsResult.Average / $goResult.Average, 2)
Write-Host "Speedup: ${speedup}x faster" -ForegroundColor Yellow
Write-Host ""

$results += [PSCustomObject]@{
    Test = "List Available (--list)"
    VBScript_Avg = $vbsResult.Average
    Go_Avg = $goResult.Average
    Speedup = "${speedup}x"
}

# Summary Table
Write-Host ""
Write-Host "============================================" -ForegroundColor Cyan
Write-Host "Summary" -ForegroundColor Cyan
Write-Host "============================================" -ForegroundColor Cyan
$results | Format-Table -AutoSize

# Calculate overall speedup
$avgSpeedup = ($results | ForEach-Object { [double]$_.Speedup.TrimEnd('x') } | Measure-Object -Average).Average
Write-Host ""
Write-Host "Average Speedup: $([math]::Round($avgSpeedup, 2))x faster" -ForegroundColor Green
Write-Host ""

# Export to CSV
$csvPath = "$PSScriptRoot\benchmark_results.csv"
$results | Export-Csv -Path $csvPath -NoTypeInformation
Write-Host "Results exported to: $csvPath" -ForegroundColor Yellow
