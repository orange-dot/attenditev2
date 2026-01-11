#Requires -RunAsAdministrator
<#
.SYNOPSIS
    Compacts Docker Desktop WSL2 virtual disk to reclaim space.

.DESCRIPTION
    Docker Desktop VHDX files don't shrink automatically after removing
    containers/images. This script compacts the disk to reclaim space.

.NOTES
    Run as Administrator in PowerShell
    Docker Desktop will be stopped during the process
#>

$ErrorActionPreference = "Stop"

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Docker Desktop Disk Compactor" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Docker VHDX locations
$vhdxPaths = @(
    "$env:LOCALAPPDATA\Docker\wsl\disk\docker_data.vhdx",
    "$env:LOCALAPPDATA\Docker\wsl\data\ext4.vhdx",
    "$env:LOCALAPPDATA\Docker\wsl\distro\ext4.vhdx"
)

# Find existing VHDX files
$foundDisks = @()
foreach ($path in $vhdxPaths) {
    if (Test-Path $path) {
        $foundDisks += $path
    }
}

# Also search for any VHDX in Docker folder
$additionalDisks = Get-ChildItem -Path "$env:LOCALAPPDATA\Docker" -Recurse -Filter "*.vhdx" -ErrorAction SilentlyContinue
foreach ($disk in $additionalDisks) {
    if ($foundDisks -notcontains $disk.FullName) {
        $foundDisks += $disk.FullName
    }
}

if ($foundDisks.Count -eq 0) {
    Write-Host "No Docker VHDX files found!" -ForegroundColor Red
    Write-Host "Checked locations:"
    foreach ($path in $vhdxPaths) {
        Write-Host "  - $path"
    }
    exit 1
}

Write-Host "Found Docker virtual disks:" -ForegroundColor Yellow
$totalBefore = 0
foreach ($disk in $foundDisks) {
    $size = (Get-Item $disk).Length
    $sizeGB = [math]::Round($size / 1GB, 2)
    $totalBefore += $size
    Write-Host "  - $disk" -ForegroundColor White
    Write-Host "    Size: $sizeGB GB" -ForegroundColor Gray
}
Write-Host ""
Write-Host "Total size before: $([math]::Round($totalBefore / 1GB, 2)) GB" -ForegroundColor Yellow
Write-Host ""

# Confirm
$confirm = Read-Host "This will stop Docker Desktop. Continue? (y/n)"
if ($confirm -ne 'y') {
    Write-Host "Cancelled." -ForegroundColor Red
    exit 0
}

Write-Host ""
Write-Host "[1/4] Stopping Docker Desktop..." -ForegroundColor Cyan

# Stop Docker Desktop
$dockerProcess = Get-Process "Docker Desktop" -ErrorAction SilentlyContinue
if ($dockerProcess) {
    Stop-Process -Name "Docker Desktop" -Force -ErrorAction SilentlyContinue
    Start-Sleep -Seconds 3
}

# Also stop related processes
$processesToStop = @("com.docker.backend", "com.docker.proxy", "Docker Desktop")
foreach ($proc in $processesToStop) {
    Stop-Process -Name $proc -Force -ErrorAction SilentlyContinue
}
Start-Sleep -Seconds 2

Write-Host "[2/4] Shutting down WSL..." -ForegroundColor Cyan
wsl --shutdown
Start-Sleep -Seconds 5

Write-Host "[3/4] Compacting virtual disks..." -ForegroundColor Cyan

foreach ($disk in $foundDisks) {
    Write-Host "  Compacting: $disk" -ForegroundColor White

    # Try Optimize-VHD first (requires Hyper-V tools)
    $hyperVAvailable = Get-Command Optimize-VHD -ErrorAction SilentlyContinue

    if ($hyperVAvailable) {
        try {
            Optimize-VHD -Path $disk -Mode Full
            Write-Host "    Done (Hyper-V)" -ForegroundColor Green
        }
        catch {
            Write-Host "    Hyper-V method failed, trying diskpart..." -ForegroundColor Yellow
            # Fallback to diskpart
            $diskpartScript = @"
select vdisk file="$disk"
attach vdisk readonly
compact vdisk
detach vdisk
exit
"@
            $diskpartScript | diskpart
            Write-Host "    Done (diskpart)" -ForegroundColor Green
        }
    }
    else {
        # Use diskpart
        Write-Host "    Using diskpart (Hyper-V tools not available)..." -ForegroundColor Yellow
        $diskpartScript = @"
select vdisk file="$disk"
attach vdisk readonly
compact vdisk
detach vdisk
exit
"@
        $diskpartScript | diskpart
        Write-Host "    Done" -ForegroundColor Green
    }
}

Write-Host ""
Write-Host "[4/4] Calculating saved space..." -ForegroundColor Cyan

$totalAfter = 0
foreach ($disk in $foundDisks) {
    $size = (Get-Item $disk).Length
    $sizeGB = [math]::Round($size / 1GB, 2)
    $totalAfter += $size
    Write-Host "  - $disk : $sizeGB GB" -ForegroundColor White
}

$saved = $totalBefore - $totalAfter
$savedGB = [math]::Round($saved / 1GB, 2)

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Results" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Before:  $([math]::Round($totalBefore / 1GB, 2)) GB" -ForegroundColor White
Write-Host "  After:   $([math]::Round($totalAfter / 1GB, 2)) GB" -ForegroundColor White
Write-Host "  Saved:   $savedGB GB" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Ask to restart Docker
$restart = Read-Host "Start Docker Desktop? (y/n)"
if ($restart -eq 'y') {
    Write-Host "Starting Docker Desktop..." -ForegroundColor Cyan
    Start-Process "$env:ProgramFiles\Docker\Docker\Docker Desktop.exe"
    Write-Host "Done!" -ForegroundColor Green
}
else {
    Write-Host "Docker Desktop not started. Start it manually when ready." -ForegroundColor Yellow
}
