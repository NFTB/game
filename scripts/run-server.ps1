Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

Push-Location "$PSScriptRoot/../server"
try {
    go run ./cmd/gameserver
}
finally {
    Pop-Location
}
