param(
  [Parameter(Mandatory)]
  [string]$ExamplePath,

  [Parameter(Mandatory)]
  [string]$OutputPath
)

$ErrorActionPreference = "Stop"

$exampleFullPath = [System.IO.Path]::GetFullPath($ExamplePath)
$outputFullPath = [System.IO.Path]::GetFullPath($OutputPath)

if (-not (Test-Path -LiteralPath $exampleFullPath -PathType Leaf)) {
  throw "environment example file does not exist"
}
if (Test-Path -LiteralPath $outputFullPath) {
  throw "refusing to overwrite an existing environment file"
}

$content = [System.IO.File]::ReadAllText($exampleFullPath)
$postgresPattern = "(?m)^POSTGRES_PASSWORD=SET_BY_INITIALIZE_LOCAL_ENV(?=\r?$)"
$sessionPattern = "(?m)^SESSION_SECRET=SET_BY_INITIALIZE_LOCAL_ENV(?=\r?$)"
if ([regex]::Matches($content, $postgresPattern).Count -ne 1) {
  throw "the PostgreSQL password sentinel must appear exactly once"
}
if ([regex]::Matches($content, $sessionPattern).Count -ne 1) {
  throw "the session secret sentinel must appear exactly once"
}

$postgresPassword = [Convert]::ToHexString([Security.Cryptography.RandomNumberGenerator]::GetBytes(32)).ToLowerInvariant()
$sessionSecret = [Convert]::ToHexString([Security.Cryptography.RandomNumberGenerator]::GetBytes(32)).ToLowerInvariant()
$content = [regex]::Replace($content, $postgresPattern, "POSTGRES_PASSWORD=$postgresPassword")
$content = [regex]::Replace($content, $sessionPattern, "SESSION_SECRET=$sessionSecret")

$outputDirectory = Split-Path -Parent $outputFullPath
if (-not (Test-Path -LiteralPath $outputDirectory -PathType Container)) {
  throw "environment output directory does not exist"
}

$encoding = [System.Text.UTF8Encoding]::new($false)
$stream = [System.IO.File]::Open($outputFullPath, [System.IO.FileMode]::CreateNew, [System.IO.FileAccess]::Write, [System.IO.FileShare]::None)
try {
  $writer = [System.IO.StreamWriter]::new($stream, $encoding)
  try {
    $writer.Write($content)
  } finally {
    $writer.Dispose()
  }
} finally {
  $stream.Dispose()
}

$outputFullPath
