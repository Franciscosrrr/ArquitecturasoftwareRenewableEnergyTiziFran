# Genera secretos locales una sola vez; .env está excluido de Git.
$ErrorActionPreference = 'Stop'
$repo = Split-Path $PSScriptRoot -Parent
$envPath = Join-Path $repo '.env'
if (-not (Test-Path -LiteralPath $envPath)) {
    $keys = @('HEALTHCHECK_TOKEN','INTERNAL_TOKEN','M2M_API_KEY','M2M_DENIED_KEY','MYSQL_ROOT_PASSWORD','MYSQL_PASSWORD')
    $lines = foreach ($key in $keys) {
        $bytes = New-Object byte[] 32
        $generator = [System.Security.Cryptography.RandomNumberGenerator]::Create()
        try { $generator.GetBytes($bytes) } finally { $generator.Dispose() }
        $value = [BitConverter]::ToString($bytes).Replace('-','').ToLowerInvariant()
        "$key=$value"
    }
    [System.IO.File]::WriteAllLines($envPath, $lines, (New-Object System.Text.UTF8Encoding($false)))
    Write-Host 'Credenciales locales generadas en .env (no se muestran ni se versionan).'
}
Push-Location $repo
try {
    & docker compose @args
    if ($LASTEXITCODE -ne 0) { throw "Docker Compose terminó con código $LASTEXITCODE" }
} finally { Pop-Location }
