param([ValidateSet('iniciar','estado','detener')][string]$Accion='estado')
$ErrorActionPreference='Stop'
$repo=Split-Path $PSScriptRoot -Parent
Push-Location $repo
try {
    if ($Accion -eq 'detener') {
        & docker compose -f compose.web-publica.yaml stop
        if ($LASTEXITCODE -ne 0) { throw 'No se pudo detener la publicación web' }
        return
    }
    if ($Accion -eq 'iniciar') {
        $gatewayId=& docker compose -f compose.yaml ps --quiet gateway
        if ($LASTEXITCODE -ne 0 -or -not $gatewayId) { throw 'Primero iniciar el sistema local con scripts/compose.ps1 up -d --no-build' }
        $gatewayHealth=(& docker inspect --format '{{.State.Health.Status}}' $gatewayId).Trim()
        if ($LASTEXITCODE -ne 0 -or $gatewayHealth -ne 'healthy') { throw 'Esperar a que el gateway local esté saludable antes de publicarlo' }
        & docker compose -f compose.web-publica.yaml up -d --no-build --wait --wait-timeout 60
        if ($LASTEXITCODE -ne 0) { throw 'No se pudo iniciar la publicación web' }
    }
    $tunnelId=& docker compose -f compose.web-publica.yaml ps --all --quiet tunnel
    if (-not $tunnelId -or $LASTEXITCODE -ne 0) { throw 'La publicación web todavía no fue iniciada' }
    $startedAt=(& docker inspect --format '{{.State.StartedAt}}' $tunnelId).Trim()
    if ($LASTEXITCODE -ne 0) { throw 'No se pudo consultar el túnel web actual' }
    $url=$null
    $attempts=if($Accion -eq 'iniciar'){45}else{1}
    for($i=0;$i -lt $attempts;$i++) {
        $log=& docker compose -f compose.web-publica.yaml logs --no-color --since $startedAt tunnel 2>&1 | Out-String
        if ($LASTEXITCODE -ne 0) { throw 'No se pudieron leer los registros del túnel web' }
        $urls=[regex]::Matches($log,'https://[a-z0-9-]+\.trycloudflare\.com')
        if ($urls.Count -gt 0) {$url=$urls[$urls.Count-1].Value;break}
        if ($Accion -eq 'iniciar') {Start-Sleep -Seconds 2}
    }
    & docker compose -f compose.web-publica.yaml ps --format '{{.Service}} {{.State}} {{.Health}}'
    $running=(& docker inspect --format '{{.State.Running}}' $tunnelId).Trim()
    if ($LASTEXITCODE -ne 0 -or $running -ne 'true') { throw 'El túnel web está detenido; su última URL no se presenta como operativa' }
    if (-not $url) { throw 'Todavía no se obtuvo la URL web; consultar scripts/publicar-web.ps1 estado' }
    $state=[ordered]@{baseUrl=$url;localUrl='http://localhost:8080';tipo='Quick Tunnel web temporal';observadoUtc=[DateTime]::UtcNow.ToString('o')}
    $runtime=Join-Path $repo '.runtime'
    New-Item -ItemType Directory -Path $runtime -Force | Out-Null
    [IO.File]::WriteAllText((Join-Path $runtime 'web-url.json'),($state | ConvertTo-Json),(New-Object Text.UTF8Encoding($false)))
    [pscustomobject]$state
} finally {Pop-Location}
