# Local and CI checks: tidy, formatting, vet, staticcheck, tests, build.
# Set $env:SKIP_STATICCHECK = "1" on Go toolchains older than 1.26.
$ErrorActionPreference = "Stop"

$StaticcheckVersion = "v0.8.2"

function Invoke-Checked {
    param([scriptblock]$Command)
    & $Command
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}

Invoke-Checked { go mod tidy }

$unformatted = gofmt -l .
if ($unformatted) {
    Write-Host "gofmt: folgende Dateien sind nicht formatiert (gofmt -w ausführen):"
    Write-Host ($unformatted -join "`n")
    exit 1
}

Invoke-Checked { go vet ./... }

if ($env:SKIP_STATICCHECK -ne "1") {
    Invoke-Checked { go run "honnef.co/go/tools/cmd/staticcheck@$StaticcheckVersion" ./... }
}

Invoke-Checked { go test ./... }
Invoke-Checked { go build -o $null . }

Write-Host "CI checks passed."
