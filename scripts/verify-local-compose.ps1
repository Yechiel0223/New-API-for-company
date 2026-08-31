param(
  [Parameter(Mandatory)]
  [string]$ComposePath,

  [Parameter(Mandatory)]
  [string]$EnvPath
)

$ErrorActionPreference = "Stop"

$composeFullPath = [System.IO.Path]::GetFullPath($ComposePath)
$envFullPath = [System.IO.Path]::GetFullPath($EnvPath)
if (-not (Test-Path -LiteralPath $composeFullPath -PathType Leaf)) {
  throw "Compose file does not exist"
}
if (-not (Test-Path -LiteralPath $envFullPath -PathType Leaf)) {
  throw "environment file does not exist"
}

$dockerCommand = Get-Command docker.exe -ErrorAction SilentlyContinue | Select-Object -First 1 -ExpandProperty Source
if (-not $dockerCommand) {
  $dockerDesktopCommand = "C:\Program Files\Docker\Docker\resources\bin\docker.exe"
  if (Test-Path -LiteralPath $dockerDesktopCommand -PathType Leaf) {
    $dockerCommand = $dockerDesktopCommand
  }
}
if (-not $dockerCommand) {
  throw "Docker CLI does not exist"
}

$renderedJson = & $dockerCommand compose --env-file $envFullPath -f $composeFullPath config --format json 2>$null
if ($LASTEXITCODE -ne 0 -or -not $renderedJson) {
  throw "Docker Compose could not render the configuration"
}

try {
  $config = ($renderedJson | Out-String) | ConvertFrom-Json -Depth 100
} catch {
  throw "Docker Compose returned invalid JSON"
}

$serviceNames = @($config.services.PSObject.Properties.Name | Sort-Object)
if (($serviceNames -join ",") -ne "new-api,postgres") {
  throw "Compose services must be exactly new-api and postgres"
}
"PASS services"

$newApi = $config.services.'new-api'
$postgres = $config.services.postgres
if ($newApi.image -ne "company/new-api:rc29-2b6f1df") {
  throw "New API image tag is not pinned"
}
if ($postgres.image -ne "postgres:15.19-alpine3.24") {
  throw "PostgreSQL image tag is not pinned"
}
if ($newApi.image -match ":latest$" -or $postgres.image -match ":latest$") {
  throw "latest image tags are forbidden"
}
"PASS pinned-images"

$newApiPorts = @($newApi.ports)
if ($newApiPorts.Count -ne 1) {
  throw "New API must publish exactly one port"
}
$newApiPort = $newApiPorts[0]
if ([string]$newApiPort.host_ip -ne "127.0.0.1" -or [int]$newApiPort.target -ne 3000 -or [int]$newApiPort.published -ne 3000) {
  throw "New API must bind only 127.0.0.1:3000 to container port 3000"
}
$postgresPorts = @($postgres.ports)
if ($postgresPorts.Count -ne 1) {
  throw "PostgreSQL must publish exactly one port for local DataGrip"
}
$postgresPort = $postgresPorts[0]
if ([string]$postgresPort.host_ip -ne "127.0.0.1" -or [int]$postgresPort.target -ne 5432 -or [int]$postgresPort.published -ne 5432) {
  throw "PostgreSQL must bind only 127.0.0.1:5432 to container port 5432"
}
"PASS ports"

$newApiEnvironment = $newApi.environment
if ([string]$newApiEnvironment.BATCH_UPDATE_ENABLED -ne "false") {
  throw "BATCH_UPDATE_ENABLED must be false"
}
if ([string]$newApiEnvironment.SQL_DSN -notmatch "^postgresql://.+@postgres:5432/.+$") {
  throw "SQL_DSN must point to the PostgreSQL service"
}
if ($newApiEnvironment.PSObject.Properties.Name -contains "REDIS_CONN_STRING") {
  throw "REDIS_CONN_STRING must not be configured"
}
"PASS database-ledger"

if ([string]$newApiEnvironment.SESSION_COOKIE_SECURE -ne "false") {
  throw "local HTTP must set SESSION_COOKIE_SECURE=false"
}
if ([string]$newApiEnvironment.TRUSTED_PROXIES -ne "none") {
  throw "the local POC must trust no reverse proxies"
}
"PASS local-http-boundary"
