$ErrorActionPreference='Stop'
& "$PSScriptRoot\compose.ps1" up -d --build
foreach ($service in @('verificar-go','verificar-real','verificar-e2e')) {
    & "$PSScriptRoot\compose.ps1" --profile verificar run --build --rm $service
}
