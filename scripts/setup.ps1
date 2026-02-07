# TermGL Development Environment Setup
# Run: .\scripts\setup.ps1

$ErrorActionPreference = "Stop"

function Write-Status($message) {
    Write-Host "[*] $message" -ForegroundColor Cyan
}

function Write-Success($message) {
    Write-Host "[+] $message" -ForegroundColor Green
}

function Write-Warning($message) {
    Write-Host "[!] $message" -ForegroundColor Yellow
}

function Write-Error($message) {
    Write-Host "[-] $message" -ForegroundColor Red
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Magenta
Write-Host "  TermGL Development Setup" -ForegroundColor Magenta
Write-Host "========================================" -ForegroundColor Magenta
Write-Host ""

# Check Go installation
Write-Status "Checking Go installation..."
if (Get-Command go -ErrorAction SilentlyContinue) {
    $goVersion = go version
    Write-Success "Go installed: $goVersion"
} else {
    Write-Error "Go is not installed. Please install Go from https://go.dev/dl/"
    exit 1
}

# Check/Install viu (terminal image viewer for Kitty)
Write-Status "Checking viu installation..."
if (Get-Command viu -ErrorAction SilentlyContinue) {
    $viuVersion = viu --version
    Write-Success "viu installed: $viuVersion"
} else {
    Write-Warning "viu not found. Installing..."

    if (Get-Command cargo -ErrorAction SilentlyContinue) {
        Write-Status "Installing viu via Cargo..."
        cargo install viu
        if ($LASTEXITCODE -eq 0) {
            Write-Success "viu installed successfully"
        } else {
            Write-Error "Failed to install viu via Cargo"
        }
    } elseif (Get-Command scoop -ErrorAction SilentlyContinue) {
        Write-Status "Installing viu via Scoop..."
        scoop install viu
        if ($LASTEXITCODE -eq 0) {
            Write-Success "viu installed successfully"
        } else {
            Write-Error "Failed to install viu via Scoop"
        }
    } else {
        Write-Warning "Cannot auto-install viu. Please install manually:"
        Write-Host "  Option 1: Install Rust, then run: cargo install viu"
        Write-Host "  Option 2: Install Scoop, then run: scoop install viu"
    }
}

# Install Go dependencies
Write-Status "Installing Go dependencies..."
Set-Location $PSScriptRoot\..
go mod download
if ($LASTEXITCODE -eq 0) {
    Write-Success "Go dependencies installed"
} else {
    Write-Error "Failed to install Go dependencies"
}

# Build the demo
Write-Status "Building demo..."
go build -o demo.exe ./examples/demo
if ($LASTEXITCODE -eq 0) {
    Write-Success "Demo built: demo.exe"
} else {
    Write-Error "Failed to build demo"
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Green
Write-Host "  Setup Complete!" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green
Write-Host ""
Write-Host "Run the demo with: .\demo.exe" -ForegroundColor White
Write-Host "Or: go run ./examples/demo" -ForegroundColor White
Write-Host ""
Write-Host "For Kitty terminal image support, viu is now available." -ForegroundColor White
Write-Host "Test with: viu <image-path>" -ForegroundColor White
Write-Host ""
