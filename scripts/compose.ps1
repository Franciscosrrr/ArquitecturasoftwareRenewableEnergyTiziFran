# Reenvía argumentos a Compose y genera una credencial efímera sin imprimirla.
$ErrorActionPreference = 'Stop'
$originalToken = $env:HEALTHCHECK_TOKEN
$repo = Split-Path $PSScriptRoot -Parent
if ([string]::IsNullOrEmpty($originalToken)) {
    $bytes = New-Object byte[] 32
    $generator = [System.Security.Cryptography.RandomNumberGenerator]::Create()
    try { $generator.GetBytes($bytes) } finally { $generator.Dispose() }
    $env:HEALTHCHECK_TOKEN = [Convert]::ToBase64String($bytes)
}
try {
    if ($env:HEALTHCHECK_TOKEN.Length -lt 32) { throw 'HEALTHCHECK_TOKEN debe tener al menos 32 caracteres' }
    Push-Location $repo
    try {
        & docker compose @args
        if ($LASTEXITCODE -ne 0) { throw "Docker Compose terminó con código $LASTEXITCODE" }
    } finally { Pop-Location }
} finally { $env:HEALTHCHECK_TOKEN = $originalToken }
