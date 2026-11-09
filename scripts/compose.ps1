# Genera secretos locales una sola vez; .env está excluido de Git.
$ErrorActionPreference = 'Stop'
if (-not $env:COMPOSE_BAKE) { $env:COMPOSE_BAKE="false" }
if (-not $env:COMPOSE_PARALLEL_LIMIT) { $env:COMPOSE_PARALLEL_LIMIT="1" }
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
if (-not (Select-String -LiteralPath $envPath -Pattern '^PANEL_PROVIDER_API_KEY=' -Quiet)) {
    $bytes = New-Object byte[] 32
    $generator = [System.Security.Cryptography.RandomNumberGenerator]::Create()
    try { $generator.GetBytes($bytes) } finally { $generator.Dispose() }
    $value = [BitConverter]::ToString($bytes).Replace('-','').ToLowerInvariant()
    [System.IO.File]::AppendAllText($envPath, "`nPANEL_PROVIDER_API_KEY=$value`n", (New-Object System.Text.UTF8Encoding($false)))
}
# Incorpora secretos nuevos sin rotar los existentes ni borrar volúmenes.
$newKeys=@('USUARIOS_DB_PASSWORD','USUARIOS_DB_ROOT_PASSWORD','CONSUMO_DB_PASSWORD','CONSUMO_DB_ROOT_PASSWORD','JWT_SECRET','ADMIN_PASSWORD','MONGO_ROOT_PASSWORD','MONGO_SOLAR_PASSWORD','RABBITMQ_PASSWORD','VALKEY_PASSWORD','SEARCH_API_KEY','GRAFANA_ADMIN_PASSWORD')
foreach ($newKey in $newKeys) {
    if (-not (Select-String -LiteralPath $envPath -Pattern ("^"+$newKey+"=") -Quiet)) {
        $randomBytes=New-Object byte[] 32
        $rng=[System.Security.Cryptography.RandomNumberGenerator]::Create()
        try { $rng.GetBytes($randomBytes) } finally { $rng.Dispose() }
        $randomValue=[BitConverter]::ToString($randomBytes).Replace('-','').ToLowerInvariant()
        [System.IO.File]::AppendAllText($envPath,"`n$newKey=$randomValue`n",(New-Object System.Text.UTF8Encoding($false)))
    }
}
$runtimeDir=Join-Path $repo '.runtime'
New-Item -ItemType Directory -Path $runtimeDir -Force | Out-Null
$healthLine=Get-Content -LiteralPath $envPath | Where-Object {$_ -match '^HEALTHCHECK_TOKEN='} | Select-Object -First 1
$metricsCredential=$healthLine.Substring('HEALTHCHECK_TOKEN='.Length).Trim().Trim('"').Trim("'")
[System.IO.File]::WriteAllText((Join-Path $runtimeDir 'metrics-token'),$metricsCredential,(New-Object System.Text.UTF8Encoding($false)))

Push-Location $repo
try {
    # RabbitMQ liga el directorio persistente al nombre del nodo; conservar el anterior.
    if (-not (Select-String -LiteralPath $envPath -Pattern '^RABBITMQ_HOSTNAME=' -Quiet)) {
        $brokerId = & docker compose ps --all --quiet rabbitmq
        if ($LASTEXITCODE -ne 0) { throw 'No se pudo consultar la identidad del broker' }
        $brokerHostname = 'renewable-broker'
        if ($brokerId) {
            $brokerHostname = (& docker inspect --format '{{.Config.Hostname}}' ($brokerId | Select-Object -First 1)).Trim()
            if ($LASTEXITCODE -ne 0 -or $brokerHostname -notmatch '^[a-zA-Z0-9.-]+$') { throw 'Identidad del broker inválida' }
        }
        [System.IO.File]::AppendAllText($envPath,"`nRABBITMQ_HOSTNAME=$brokerHostname`n",(New-Object System.Text.UTF8Encoding($false)))
    }
    # Construir cada imagen por separado evita que Bake paralelice todo el lote.
    $defaultBuildServices=@('consumo','electrodomesticos','usuarios','solar','solar-2','proveedor-paneles-sim','gateway')
    $buildable=$defaultBuildServices+@('verificar-go','verificar-e2e','mock','verificar')
    $commandIndex=-1
    for($argumentIndex=0;$argumentIndex -lt $args.Count;$argumentIndex++) {
        if($args[$argumentIndex] -in @('up','build','run','stop','start','down','ps','exec','logs','restart','config','pull')){$commandIndex=$argumentIndex;break}
    }
    if($commandIndex -ge 0 -and ($args[$commandIndex] -eq 'build' -or ($args[$commandIndex] -eq 'up' -and $args -contains '--build'))) {
        $globalArgs=@();if($commandIndex -gt 0){$globalArgs=@($args[0..($commandIndex-1)])}
        $commandArgs=@($args[($commandIndex+1)..($args.Count-1)])
        if($commandIndex -eq $args.Count-1){$commandArgs=@()}
        $targets=@($commandArgs | Where-Object {$_ -in $buildable})
        if($targets.Count -eq 0){$targets=$defaultBuildServices}
        $buildOptions=@()
        if($args[$commandIndex] -eq 'build'){$buildOptions=@($commandArgs | Where-Object {$_ -notin $buildable})}
        foreach($buildTarget in $targets){
            & docker compose @globalArgs build @buildOptions $buildTarget
            if($LASTEXITCODE -ne 0){throw "Falló la compilación de $buildTarget"}
        }
        if($args[$commandIndex] -eq 'build'){return}
        $args=@($args | Where-Object {$_ -ne '--build'})
    }
    if($commandIndex -ge 0 -and $args[$commandIndex] -eq 'run' -and $args -contains '--build'){
        $testTarget=@($args | Where-Object {$_ -in $buildable} | Select-Object -Last 1)
        if($testTarget.Count -eq 1){
            $globalArgs=@();if($commandIndex -gt 0){$globalArgs=@($args[0..($commandIndex-1)])}
            & docker compose @globalArgs build $testTarget[0]
            if($LASTEXITCODE -ne 0){throw 'Falló la compilación de verificación'}
            $args=@($args | Where-Object {$_ -ne '--build'})
        }
    }
    & docker compose @args
    if ($LASTEXITCODE -ne 0) { throw "Docker Compose terminó con código $LASTEXITCODE" }
} finally { Pop-Location }
