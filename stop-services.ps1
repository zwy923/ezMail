# Stop all running services on ports 8080, 8081, 8000

Write-Host "Checking for processes on ports 8080, 8081, 8000..." -ForegroundColor Yellow

# Function to kill process on a port
function Stop-ProcessOnPort {
    param([int]$Port)
    
    $connections = netstat -ano | findstr ":$Port"
    if ($connections) {
        $processIds = $connections | ForEach-Object {
            if ($_ -match '\s+(\d+)$') {
                $matches[1]
            }
        } | Select-Object -Unique
        
        foreach ($processId in $processIds) {
            try {
                $process = Get-Process -Id $processId -ErrorAction Stop
                Write-Host "Stopping process $processId ($($process.ProcessName)) on port $Port" -ForegroundColor Red
                Stop-Process -Id $processId -Force
                Write-Host "Process $processId stopped successfully" -ForegroundColor Green
            } catch {
                Write-Host "Could not stop process $processId : $_" -ForegroundColor Yellow
            }
        }
    } else {
        Write-Host "No process found on port $Port" -ForegroundColor Green
    }
}

Stop-ProcessOnPort -Port 8080
Stop-ProcessOnPort -Port 8081
Stop-ProcessOnPort -Port 8000

Write-Host "`nDone! Ports should be free now." -ForegroundColor Green
