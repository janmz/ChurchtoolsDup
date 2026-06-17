$ErrorActionPreference = "Stop"

go mod tidy
go vet ./...
go test ./...
go build -o $null .

Write-Host "CI checks passed."
