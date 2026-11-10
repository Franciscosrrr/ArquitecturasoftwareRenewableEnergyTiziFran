param([Parameter(Mandatory=$true)][string]$BaseUrl)
$ErrorActionPreference='Stop'
if (-not $BaseUrl.StartsWith('https://')) { throw 'Se requiere HTTPS público.' }
if (-not $env:M2M_API_KEY -or -not $env:M2M_DENIED_KEY) { throw 'Cargar las claves en M2M_API_KEY y M2M_DENIED_KEY sin imprimirlas.' }
$endpoint=$BaseUrl.TrimEnd('/')+'/api/v1/estimaciones-consumo'
$body='{"diasPeriodo":30,"equipos":[{"id":"prueba-publicacion","modo":"POTENCIA","cantidad":1,"potenciaW":100,"horasPorDia":5,"diasUso":30,"factorFuncionamiento":1}]}'
$watch=[Diagnostics.Stopwatch]::StartNew()
$result=Invoke-WebRequest -Uri $endpoint -Method Post -ContentType 'application/json' -Headers @{'X-API-Key'=$env:M2M_API_KEY} -Body $body -SkipHttpErrorCheck -TimeoutSec 90
$watch.Stop()
if ($result.StatusCode -ne 200) { throw "Respuesta esperada 200; recibida $($result.StatusCode)" }
foreach($test in @(@{key='';expected=401},@{key=$env:M2M_DENIED_KEY;expected=403})) {
 $headers=@{}; if($test.key){$headers['X-API-Key']=$test.key}
 $response=Invoke-WebRequest -Uri $endpoint -Method Post -ContentType 'application/json' -Headers $headers -Body $body -SkipHttpErrorCheck
 if($response.StatusCode -ne $test.expected){throw "Estado esperado $($test.expected); recibido $($response.StatusCode)"}
}
$contract=Invoke-RestMethod -Uri ($BaseUrl.TrimEnd('/')+'/openapi.json')
if($contract.info.version -ne '1.0.1'){throw 'Versión de contrato inesperada.'}
[pscustomobject]@{url=$endpoint;contrato=$contract.info.version;latenciaMs=$watch.ElapsedMilliseconds;dentroDe5s=$watch.ElapsedMilliseconds -le 5000;dentroDe1200ms=$watch.ElapsedMilliseconds -le 1200;verificadoUtc=[DateTime]::UtcNow.ToString('o')}
