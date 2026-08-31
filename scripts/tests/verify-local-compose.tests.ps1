$ErrorActionPreference = "Stop"

function Assert-True {
  param(
    [Parameter(Mandatory)]
    [bool]$Condition,

    [Parameter(Mandatory)]
    [string]$Message
  )

  if (-not $Condition) {
    throw $Message
  }
}

$scriptPath = Join-Path (Split-Path -Parent $PSScriptRoot) "verify-local-compose.ps1"
Assert-True -Condition (Test-Path -LiteralPath $scriptPath) -Message "verify-local-compose.ps1 must exist"

$repoRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$composePath = Join-Path $repoRoot "deploy\compose.yaml"
$envPath = Join-Path $repoRoot "deploy\.env"
Assert-True -Condition (Test-Path -LiteralPath $composePath) -Message "deploy/compose.yaml must exist"
Assert-True -Condition (Test-Path -LiteralPath $envPath) -Message "deploy/.env must exist"

$renderedConfig = docker compose --env-file $envPath -f $composePath config --format json | ConvertFrom-Json -Depth 100
$postgresPorts = @($renderedConfig.services.postgres.ports)
Assert-True -Condition ($postgresPorts.Count -eq 1) -Message "PostgreSQL must publish exactly one port for local DataGrip"
$postgresPort = $postgresPorts[0]
Assert-True -Condition (
  [string]$postgresPort.host_ip -eq "127.0.0.1" -and
  [int]$postgresPort.target -eq 5432 -and
  [int]$postgresPort.published -eq 5432
) -Message "PostgreSQL must bind only 127.0.0.1:5432 to container port 5432"

$validOutput = & $scriptPath -ComposePath $composePath -EnvPath $envPath
Assert-True -Condition ($LASTEXITCODE -eq 0) -Message "the approved local Compose config must pass"
Assert-True -Condition (-not (($validOutput | Out-String) -match "POSTGRES_PASSWORD|SESSION_SECRET|SQL_DSN")) -Message "verification output must not reveal environment values"

$tempRoot = [System.IO.Path]::GetFullPath([System.IO.Path]::GetTempPath())
$testDirectory = Join-Path $tempRoot ("new-api-compose-test-" + [guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Path $testDirectory | Out-Null

try {
  $invalidComposePath = Join-Path $testDirectory "compose-with-redis.yaml"
  $validCompose = Get-Content -Raw -LiteralPath $composePath
  $invalidCompose = $validCompose -replace "(?m)^services:\s*$", "services:`n  redis:`n    image: redis:7-alpine"
  $invalidCompose | Set-Content -LiteralPath $invalidComposePath -Encoding utf8NoBOM

  $redisAccepted = $false
  try {
    & $scriptPath -ComposePath $invalidComposePath -EnvPath $envPath | Out-Null
    $redisAccepted = $true
  } catch {
    $redisAccepted = $false
  }
  Assert-True -Condition (-not $redisAccepted) -Message "a Compose config containing a Redis service must be rejected"

  "PASS verify-local-compose behavior"
} finally {
  $resolvedTestDirectory = [System.IO.Path]::GetFullPath($testDirectory)
  if (-not $resolvedTestDirectory.StartsWith($tempRoot, [System.StringComparison]::OrdinalIgnoreCase)) {
    throw "refusing to remove a directory outside the system temp root"
  }
  Remove-Item -LiteralPath $resolvedTestDirectory -Recurse -Force
}
