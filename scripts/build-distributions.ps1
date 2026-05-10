param(
    [string]$Version = "",
    [string]$Wails = "",
    [string]$OutputDir = "",
    [string[]]$DesktopPlatforms = @("windows/amd64", "darwin/universal", "linux/amd64"),
    [switch]$SkipDesktop,
    [switch]$SkipChecks,
    [switch]$Clean
)

$ErrorActionPreference = "Stop"

function Write-Step {
    param([string]$Message)
    Write-Host ""
    Write-Host "==> $Message" -ForegroundColor Cyan
}

function Invoke-Checked {
    param(
        [string]$FilePath,
        [string[]]$Arguments,
        [string]$WorkingDirectory
    )
    $previous = Get-Location
    try {
        Set-Location $WorkingDirectory
        & $FilePath @Arguments
        if ($LASTEXITCODE -ne 0) {
            throw "$FilePath $($Arguments -join ' ') failed with exit code $LASTEXITCODE"
        }
    }
    finally {
        Set-Location $previous
    }
}

function Resolve-Wails {
    param([string]$Requested)
    if ($Requested) {
        return $Requested
    }
    if ($env:WAILS) {
        return $env:WAILS
    }
    $cmd = Get-Command wails -ErrorAction SilentlyContinue
    if ($cmd) {
        return $cmd.Source
    }
    $gopath = (& go env GOPATH).Trim()
    if ($gopath) {
        $candidate = Join-Path $gopath "bin\wails.exe"
        if (Test-Path $candidate) {
            return $candidate
        }
    }
    return ""
}

function Safe-RemoveDirectory {
    param(
        [string]$Path,
        [string]$ExpectedParent
    )
    if (-not (Test-Path $Path)) {
        return
    }
    $resolvedPath = [System.IO.Path]::GetFullPath($Path)
    $resolvedParent = [System.IO.Path]::GetFullPath($ExpectedParent)
    if (-not $resolvedPath.StartsWith($resolvedParent, [System.StringComparison]::OrdinalIgnoreCase)) {
        throw "Refusing to delete outside expected parent: $resolvedPath"
    }
    Remove-Item -LiteralPath $resolvedPath -Recurse -Force
}

function New-StageReadme {
    param(
        [string]$StageDir,
        [string]$Name,
        [string]$DesktopNote
    )
    @"
Cadenza $Name distribution

CLI:
- Run the cadenza binary from this folder.
- Offline example: cadenza --bpm 122 --key Am --no-llm

Desktop:
$DesktopNote

Output:
- CLI default output is ./output unless another output path is selected.
- Desktop default output is ~/cadenza-output.
"@ | Set-Content -Path (Join-Path $StageDir "DISTRIBUTION.txt") -NoNewline
}

function Assert-PackageContents {
    param(
        [string]$StageDir,
        [string]$PackageName,
        [bool]$DesktopExpected
    )

    $isWindows = $PackageName -like "windows-*"
    $cliName = if ($isWindows) { "cadenza.exe" } else { "cadenza" }
    if ($PackageName -eq "macos-universal") {
        $cliCandidates = @("cadenza-darwin-amd64", "cadenza-darwin-arm64")
        foreach ($candidate in $cliCandidates) {
            if (-not (Test-Path (Join-Path $StageDir $candidate))) {
                throw "Package $PackageName is missing CLI binary: $candidate"
            }
        }
    }
    elseif (-not (Test-Path (Join-Path $StageDir $cliName))) {
        throw "Package $PackageName is missing CLI binary: $cliName"
    }

    if (-not $DesktopExpected) {
        return
    }

    $desktopDir = Join-Path $StageDir "desktop"
    if (-not (Test-Path $desktopDir)) {
        throw "Package $PackageName is missing desktop folder."
    }

    if ($isWindows) {
        $desktopExe = Join-Path $desktopDir "cadenza-desktop.exe"
        if (-not (Test-Path $desktopExe)) {
            throw "Windows package must contain desktop\cadenza-desktop.exe."
        }
    }
}

function Build-Cli {
    param(
        [string]$GOOS,
        [string]$GOARCH,
        [string]$OutFile,
        [string]$BackendDir
    )
    $oldGOOS = $env:GOOS
    $oldGOARCH = $env:GOARCH
    $oldCGO = $env:CGO_ENABLED
    try {
        $env:GOOS = $GOOS
        $env:GOARCH = $GOARCH
        $env:CGO_ENABLED = "0"
        Invoke-Checked -FilePath "go" -Arguments @("build", "-ldflags=-s -w", "-o", $OutFile, "./cmd/cadenza/") -WorkingDirectory $BackendDir
    }
    finally {
        $env:GOOS = $oldGOOS
        $env:GOARCH = $oldGOARCH
        $env:CGO_ENABLED = $oldCGO
    }
}

function Try-BuildDesktop {
    param(
        [string]$Platform,
        [string]$WailsPath,
        [string]$DesktopDir,
        [string]$StageDir
    )
    if (-not $WailsPath) {
        return "Desktop app not included: Wails CLI was not found. Install it with go install github.com/wailsapp/wails/v2/cmd/wails@latest or pass -Wails <path>."
    }

    $binDir = Join-Path $DesktopDir "build\bin"
    try {
        Safe-RemoveDirectory -Path $binDir -ExpectedParent $DesktopDir
    }
    catch {
        if (Test-Path $binDir) {
            $desktopStage = Join-Path $StageDir "desktop"
            New-Item -ItemType Directory -Force -Path $desktopStage | Out-Null
            Copy-Item -Path (Join-Path $binDir "*") -Destination $desktopStage -Recurse -Force
            return "Desktop app included from existing build/bin for ${Platform}; it was not rebuilt because build/bin is locked. Close any running cadenza-desktop app for a fresh rebuild. $($_.Exception.Message)"
        }
        return "Desktop app not included for ${Platform}: could not clean build/bin. Close any running cadenza-desktop app and try again. $($_.Exception.Message)"
    }

    try {
        Invoke-Checked -FilePath $WailsPath -Arguments @("build", "-platform", $Platform) -WorkingDirectory $DesktopDir
    }
    catch {
        return "Desktop app not included for ${Platform}: $($_.Exception.Message)"
    }

    if (-not (Test-Path $binDir)) {
        return "Desktop app not included for ${Platform}: Wails did not create build/bin."
    }

    $desktopStage = Join-Path $StageDir "desktop"
    New-Item -ItemType Directory -Force -Path $desktopStage | Out-Null
    Copy-Item -Path (Join-Path $binDir "*") -Destination $desktopStage -Recurse -Force
    return "Desktop app included from Wails platform $Platform."
}

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$backendDir = Join-Path $repoRoot "backend"
$desktopDir = Join-Path $backendDir "cmd\desktop"

if (-not $OutputDir) {
    $OutputDir = Join-Path $repoRoot "dist"
}
$distDir = [System.IO.Path]::GetFullPath($OutputDir)
$stageRoot = Join-Path $distDir "_stage"

if (-not $Version) {
    try {
        $Version = (& git -C $repoRoot describe --tags --always --dirty 2>$null).Trim()
    }
    catch {
        $Version = "dev"
    }
    if (-not $Version) {
        $Version = "dev"
    }
}

if ($Clean) {
    Safe-RemoveDirectory -Path $distDir -ExpectedParent $repoRoot
}
New-Item -ItemType Directory -Force -Path $distDir | Out-Null
Safe-RemoveDirectory -Path $stageRoot -ExpectedParent $distDir
New-Item -ItemType Directory -Force -Path $stageRoot | Out-Null

Write-Step "Building Cadenza distributions ($Version)"
Write-Host "Repo: $repoRoot"
Write-Host "Dist: $distDir"

if (-not $SkipChecks) {
    Write-Step "Running Go tests"
    Invoke-Checked -FilePath "go" -Arguments @("test", "./...") -WorkingDirectory $backendDir
}

$wailsPath = Resolve-Wails -Requested $Wails
if (-not $SkipDesktop) {
    if ($wailsPath) {
        Write-Host "Wails: $wailsPath"
    }
    else {
        Write-Warning "Wails CLI not found. Desktop binaries will be skipped."
    }
}

$packages = @(
    @{
        Name = "windows-amd64"
        Stage = "windows"
        Zip = "cadenza-$Version-windows-amd64.zip"
        Cli = @(@{ GOOS = "windows"; GOARCH = "amd64"; Out = "cadenza.exe" })
        DesktopPlatform = "windows/amd64"
    },
    @{
        Name = "macos-universal"
        Stage = "macos"
        Zip = "cadenza-$Version-macos-universal.zip"
        Cli = @(
            @{ GOOS = "darwin"; GOARCH = "amd64"; Out = "cadenza-darwin-amd64" },
            @{ GOOS = "darwin"; GOARCH = "arm64"; Out = "cadenza-darwin-arm64" }
        )
        DesktopPlatform = "darwin/universal"
    },
    @{
        Name = "linux-amd64"
        Stage = "linux"
        Zip = "cadenza-$Version-linux-amd64.zip"
        Cli = @(@{ GOOS = "linux"; GOARCH = "amd64"; Out = "cadenza" })
        DesktopPlatform = "linux/amd64"
    }
)

foreach ($pkg in $packages) {
    Write-Step "Packaging $($pkg.Name)"
    $stageDir = Join-Path $stageRoot $pkg.Stage
    New-Item -ItemType Directory -Force -Path $stageDir | Out-Null

    foreach ($cli in $pkg.Cli) {
        $outFile = Join-Path $stageDir $cli.Out
        Write-Host "CLI $($cli.GOOS)/$($cli.GOARCH) -> $outFile"
        Build-Cli -GOOS $cli.GOOS -GOARCH $cli.GOARCH -OutFile $outFile -BackendDir $backendDir
    }

    $desktopNote = "Desktop app not requested."
    if (-not $SkipDesktop) {
        if ($DesktopPlatforms -contains $pkg.DesktopPlatform) {
            Write-Host "Desktop $($pkg.DesktopPlatform)"
            $desktopNote = Try-BuildDesktop -Platform $pkg.DesktopPlatform -WailsPath $wailsPath -DesktopDir $desktopDir -StageDir $stageDir
            Write-Host $desktopNote
        }
        else {
            $desktopNote = "Desktop app skipped by -DesktopPlatforms."
        }
    }

    New-StageReadme -StageDir $stageDir -Name $pkg.Name -DesktopNote $desktopNote
    Copy-Item -Path (Join-Path $repoRoot "README.md") -Destination $stageDir -Force
    Copy-Item -Path (Join-Path $repoRoot "LICENSE") -Destination $stageDir -Force

    $desktopExpected = $desktopNote -like "Desktop app included*"
    Assert-PackageContents -StageDir $stageDir -PackageName $pkg.Name -DesktopExpected $desktopExpected

    $zipPath = Join-Path $distDir $pkg.Zip
    if (Test-Path $zipPath) {
        Remove-Item -LiteralPath $zipPath -Force
    }
    Compress-Archive -Path (Join-Path $stageDir "*") -DestinationPath $zipPath -Force
    Write-Host "Created $zipPath" -ForegroundColor Green
}

Write-Step "Done"
Get-ChildItem -Path $distDir -Filter "*.zip" | Select-Object Name, Length, LastWriteTime
