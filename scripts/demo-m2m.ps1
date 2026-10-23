$ErrorActionPreference='Stop'
$repo=Split-Path $PSScriptRoot -Parent
$envFile=Join-Path $repo '.env'
if (-not (Test-Path -LiteralPath $envFile)) {throw 'Primero iniciar con scripts/compose.ps1 up -d --build'}
$line=Get-Content -LiteralPath $envFile | Where-Object { $_.StartsWith('M2M_API_KEY=') } | Select-Object -First 1
if (-not $line) {throw 'Falta M2M_API_KEY en el entorno local'}
$apiKey=$line.Substring('M2M_API_KEY='.Length)
$requestBody=Get-Content -LiteralPath (Join-Path $repo 'docs\contracts\v1\examples\solicitud-valida.json') -Raw
$result=Invoke-RestMethod -Uri 'http://localhost:8080/api/v1/estimaciones-consumo' -Method Post -ContentType 'application/json' -Headers @{'X-API-Key'=$apiKey;'X-Request-Id'='demo-entrega2'} -Body $requestBody
$result | ConvertTo-Json -Depth 8
