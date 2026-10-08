$ErrorActionPreference = 'Stop'
$repo = Split-Path $PSScriptRoot -Parent
Push-Location $repo
try {
 python -m unittest discover -s mock -p "test_*.py" -v
 if ($LASTEXITCODE -ne 0) { throw 'Fallaron las pruebas del mock' }
 foreach ($component in @('gateway','services/usuarios','services/electrodomesticos','services/consumo','services/solar')) {
  Push-Location $component
  try { go test ./...; if ($LASTEXITCODE -ne 0) { throw "Falló $component" } } finally { Pop-Location }
 }
} finally { Pop-Location }
