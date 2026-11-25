<#
.SYNOPSIS
    Installe Verk sous Windows.

.DESCRIPTION
    Télécharge l'installeur Verk-windows-x64-setup.exe depuis la GitHub Release,
    vérifie son empreinte SHA-256 puis l'exécute en mode silencieux. Verk est
    installé pour l'utilisateur courant dans %LOCALAPPDATA%\Programs\Verk, sans
    droits administrateur, avec ses raccourcis (menu Démarrer et bureau).

    Le dépôt est privé : le téléchargement passe par la CLI gh (connectée),
    sinon par un jeton $env:GITHUB_TOKEN (ou $env:GH_TOKEN), sinon par les
    liens publics.

.PARAMETER Version
    Version à installer (v3.0.0 ou 3.0.0). Par défaut : la dernière publiée.

.PARAMETER Launch
    Lance Verk une fois l'installation terminée.

.EXAMPLE
    powershell -ExecutionPolicy Bypass -File scripts\install.ps1
.EXAMPLE
    powershell -ExecutionPolicy Bypass -File scripts\install.ps1 -Version v3.0.0 -Launch
#>
[CmdletBinding()]
param(
    [string]$Version = $env:VERK_VERSION,
    [switch]$Launch
)

$ErrorActionPreference = 'Stop'
# Invoke-WebRequest est très lent avec la barre de progression sous PowerShell 5.1.
$ProgressPreference = 'SilentlyContinue'
[Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12

$Repo = 'verkapp/verk'
$App = 'Verk'
$Asset = "$App-windows-x64-setup.exe"

if ([string]::IsNullOrWhiteSpace($Version)) { $Version = 'latest' }
if ($Version -ne 'latest' -and -not $Version.StartsWith('v')) { $Version = "v$Version" }
$Token = if ($env:GITHUB_TOKEN) { $env:GITHUB_TOKEN } else { $env:GH_TOKEN }

function Write-Step([string]$Message) { Write-Host "==> $Message" -ForegroundColor Cyan }

function Test-GhReady {
    if (-not (Get-Command gh -ErrorAction SilentlyContinue)) { return $false }
    & gh auth status *> $null
    return $LASTEXITCODE -eq 0
}

$script:Release = $null
function Get-Release {
    if ($null -eq $script:Release) {
        $path = if ($Version -eq 'latest') { 'releases/latest' } else { "releases/tags/$Version" }
        $headers = @{ Authorization = "Bearer $Token"; Accept = 'application/vnd.github+json' }
        try {
            $script:Release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/$path" -Headers $headers
        } catch {
            throw "Release $Version introuvable sur $Repo (jeton valide ?) : $($_.Exception.Message)"
        }
    }
    return $script:Release
}

function Get-ReleaseFile([string]$Name, [string]$Directory) {
    $target = Join-Path $Directory $Name
    if (Test-GhReady) {
        $ghArgs = @('release', 'download')
        if ($Version -ne 'latest') { $ghArgs += $Version }
        $ghArgs += @('--repo', $Repo, '--pattern', $Name, '--dir', $Directory, '--clobber')
        & gh @ghArgs
        if ($LASTEXITCODE -ne 0) { throw "Téléchargement de $Name impossible avec gh." }
    } elseif ($Token) {
        $match = (Get-Release).assets | Where-Object { $_.name -eq $Name } | Select-Object -First 1
        if (-not $match) { throw "$Name absent de la release." }
        $headers = @{ Authorization = "Bearer $Token"; Accept = 'application/octet-stream' }
        Invoke-WebRequest -Uri $match.url -Headers $headers -OutFile $target -UseBasicParsing
    } else {
        $url = if ($Version -eq 'latest') {
            "https://github.com/$Repo/releases/latest/download/$Name"
        } else {
            "https://github.com/$Repo/releases/download/$Version/$Name"
        }
        try {
            Invoke-WebRequest -Uri $url -OutFile $target -UseBasicParsing
        } catch {
            throw "Téléchargement de $Name impossible. Le dépôt est privé : connecte gh (gh auth login) ou définis `$env:GITHUB_TOKEN."
        }
    }
    if (-not (Test-Path $target)) { throw "$Name n'a pas été téléchargé." }
    return $target
}

if (-not [Environment]::Is64BitOperatingSystem) {
    throw "$App nécessite Windows 64 bits."
}

$work = Join-Path ([IO.Path]::GetTempPath()) ("verk-install-" + [Guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $work | Out-Null
try {
    Write-Step "Téléchargement de $Asset ($Version)…"
    $setup = Get-ReleaseFile $Asset $work
    $sumFile = Get-ReleaseFile "$Asset.sha256" $work

    $expected = ((Get-Content $sumFile -Raw).Trim() -split '\s+')[0].ToLowerInvariant()
    $actual = (Get-FileHash $setup -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($expected -ne $actual) { throw "Empreinte SHA-256 invalide pour $Asset." }
    Write-Step 'Empreinte SHA-256 vérifiée.'

    $running = Get-Process -Name $App -ErrorAction SilentlyContinue
    if ($running) {
        Write-Step "Fermeture de $App…"
        $running | Stop-Process -Force
        Start-Sleep -Seconds 1
    }

    Write-Step 'Installation…'
    $process = Start-Process -FilePath $setup -ArgumentList '/S' -Wait -PassThru
    if ($process.ExitCode -ne 0) { throw "L'installeur a échoué (code $($process.ExitCode))." }

    $exe = Join-Path $env:LOCALAPPDATA "Programs\$App\$App.exe"
    if (-not (Test-Path $exe)) { throw "Installation terminée mais $exe est introuvable." }
    Write-Step "$App est installé : $exe"

    if ($Launch) { Start-Process -FilePath $exe }
} finally {
    Remove-Item -Recurse -Force $work -ErrorAction SilentlyContinue
}
