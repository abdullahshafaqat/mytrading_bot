$env:LOG_FILE = ""
$job = Start-Job -ScriptBlock { Set-Location "d:\trading bot\backend"; go run ./cmd/bot }
Start-Sleep -Seconds 20

Write-Host "`n=== GET /paper/export.csv ==="
Invoke-RestMethod -Uri http://localhost:8080/paper/export.csv

Stop-Job -Job $job
