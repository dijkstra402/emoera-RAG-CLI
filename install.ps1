$ErrorActionPreference = 'Stop'

$Repository = if ($env:EMOERA_INSTALL_REPOSITORY) { $env:EMOERA_INSTALL_REPOSITORY } else { 'dijkstra402/emoera-RAG-CLI' }
$InstallDir = if ($env:EMOERA_INSTALL_DIR) { $env:EMOERA_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA 'Programs\Emoera' }

$architecture = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString().ToLowerInvariant()
if ($architecture -ne 'x64') {
    throw "暂不支持当前 Windows 架构: $architecture"
}

if ($env:EMOERA_VERSION) {
    $tag = $env:EMOERA_VERSION
} else {
    $release = Invoke-RestMethod -Headers @{ 'User-Agent' = 'emoera-installer' } -Uri "https://api.github.com/repos/$Repository/releases/latest"
    $tag = $release.tag_name
}

$version = $tag.TrimStart('v')
$archive = "emoera-cli_${version}_windows_amd64.zip"
$baseUrl = "https://github.com/$Repository/releases/download/$tag"
$tempDir = Join-Path ([System.IO.Path]::GetTempPath()) ("emoera-install-" + [guid]::NewGuid())

try {
    New-Item -ItemType Directory -Force -Path $tempDir | Out-Null
    $archivePath = Join-Path $tempDir $archive
    $checksumPath = Join-Path $tempDir 'SHA256SUMS'
    Invoke-WebRequest -UseBasicParsing -Uri "$baseUrl/$archive" -OutFile $archivePath
    Invoke-WebRequest -UseBasicParsing -Uri "$baseUrl/SHA256SUMS" -OutFile $checksumPath

    $checksumLine = Get-Content $checksumPath | Where-Object { $_ -match "\s$([regex]::Escape($archive))$" } | Select-Object -First 1
    if (-not $checksumLine) { throw "SHA256SUMS 中没有找到 $archive" }
    $expected = ($checksumLine -split '\s+')[0].ToLowerInvariant()
    $actual = (Get-FileHash -Algorithm SHA256 $archivePath).Hash.ToLowerInvariant()
    if ($actual -ne $expected) { throw '安装包 SHA-256 校验失败' }

    Expand-Archive -Path $archivePath -DestinationPath $tempDir -Force
    New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
    Copy-Item (Join-Path $tempDir "emoera-cli_${version}_windows_amd64\emoera.exe") (Join-Path $InstallDir 'emoera.exe') -Force

    $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    $pathEntries = @($userPath -split ';' | Where-Object { $_ })
    if ($pathEntries -notcontains $InstallDir) {
        [Environment]::SetEnvironmentVariable('Path', (($pathEntries + $InstallDir) -join ';'), 'User')
    }
    if (($env:Path -split ';') -notcontains $InstallDir) { $env:Path = "$InstallDir;$env:Path" }

    Write-Host "安装完成: $InstallDir\emoera.exe"
    & (Join-Path $InstallDir 'emoera.exe') --version
    Write-Host '如果其他 PowerShell 窗口还找不到 emoera，请重新打开终端。'
} finally {
    Remove-Item -Recurse -Force $tempDir -ErrorAction SilentlyContinue
}
