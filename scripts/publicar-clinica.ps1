param([ValidateSet('iniciar','estado','detener')][string]$Accion='estado')
$ErrorActionPreference='Stop'
$repo=Split-Path $PSScriptRoot -Parent
$envFile=Join-Path $repo '.runtime/clinica.env'
Push-Location $repo
try {
    if ($Accion -eq 'detener') {
        & docker compose -f compose.publicacion.yaml stop
        if ($LASTEXITCODE -ne 0) { throw 'No se pudo detener la publicación' }
        return
    }
    if ($Accion -eq 'iniciar') {
        New-Item -ItemType Directory -Path (Split-Path $envFile -Parent) -Force | Out-Null
        if (-not (Test-Path -LiteralPath $envFile)) {
            $lines=foreach($key in @('M2M_API_KEY','M2M_DENIED_KEY','HEALTHCHECK_TOKEN','INTERNAL_TOKEN')) {
                $bytes=New-Object byte[] 32
                $rng=[Security.Cryptography.RandomNumberGenerator]::Create()
                try {$rng.GetBytes($bytes)} finally {$rng.Dispose()}
                "$key=$([BitConverter]::ToString($bytes).Replace('-','').ToLowerInvariant())"
            }
            [IO.File]::WriteAllLines($envFile,$lines,(New-Object Text.UTF8Encoding($false)))
        }
        $imageId=& docker image ls --quiet renewable-cloud-final:latest
        if ($LASTEXITCODE -ne 0) { throw 'No se pudo consultar la imagen de publicación' }
        if (-not $imageId) {
            & docker compose -f compose.publicacion.yaml build api-publica
            if ($LASTEXITCODE -ne 0) { throw 'No se pudo construir la API M2M' }
        }
        & docker compose -f compose.publicacion.yaml up -d --no-build
        if ($LASTEXITCODE -ne 0) { throw 'No se pudo iniciar la publicación' }
    }
    if (-not (Test-Path -LiteralPath $envFile)) { throw 'La publicación todavía no fue iniciada' }
    $tunnelId=& docker compose -f compose.publicacion.yaml ps --all --quiet tunnel
    if (-not $tunnelId -or $LASTEXITCODE -ne 0) { throw 'No existe un contenedor del túnel' }
    $startedAt=(& docker inspect --format '{{.State.StartedAt}}' $tunnelId).Trim()
    if ($LASTEXITCODE -ne 0) { throw 'No se pudo consultar la ejecución actual del túnel' }
    $url=$null
    $attempts=if($Accion -eq 'iniciar'){45}else{1}
    for($i=0;$i -lt $attempts;$i++) {
        $log=& docker compose -f compose.publicacion.yaml logs --no-color --since $startedAt tunnel 2>&1 | Out-String
        $urls=[regex]::Matches($log,'https://[a-z0-9-]+\.trycloudflare\.com')
        if ($urls.Count -gt 0) {$url=$urls[$urls.Count-1].Value;break}
        if ($Accion -eq 'iniciar') {Start-Sleep -Seconds 2}
    }
    & docker compose -f compose.publicacion.yaml ps --format '{{.Service}} {{.State}} {{.Health}}'
    $running=(& docker inspect --format '{{.State.Running}}' $tunnelId).Trim()
    if ($running -ne 'true') { throw 'El túnel está detenido; no se presenta su última URL como operativa' }
    if ($url) {
        $state=[ordered]@{baseUrl=$url;endpoint="$url/api/v1/estimaciones-consumo";openapi="$url/openapi.json";tipo='Quick Tunnel temporal';observadoUtc=[DateTime]::UtcNow.ToString('o')}
        [IO.File]::WriteAllText((Join-Path $repo '.runtime/clinica-url.json'),($state | ConvertTo-Json),(New-Object Text.UTF8Encoding($false)))
        [pscustomobject]$state
    } else {throw 'Todavía no se obtuvo una URL del túnel; consultar sus logs sin habilitar debug'}
} finally {Pop-Location}
