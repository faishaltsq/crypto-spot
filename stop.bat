@echo off
setlocal enabledelayedexpansion
cd /d "%~dp0"

echo ============================================
echo   Crypto Spot Signal - STOPPING (FULL CLEAN)
echo ============================================

echo.
echo [1/3] Stopping frontend...
REM Close the dev window opened by start.bat (title set in start.bat)
taskkill /FI "WINDOWTITLE eq crypto-spot-signal-web*" /T /F >nul 2>&1
REM Kill whatever is still LISTENING on the frontend port 3000 (stray next dev)
for /f "tokens=5" %%p in ('netstat -ano ^| findstr ":3000" ^| findstr "LISTENING"') do (
  taskkill /PID %%p /F >nul 2>&1
)
echo   Frontend stopped.

echo.
echo [2/3] Checking Docker daemon...
docker info >nul 2>&1
if errorlevel 1 (
  echo   Docker daemon is not running - backend containers are already down.
  echo   Skipping docker teardown.
  goto done
)

echo.
echo [3/3] Tearing down FULL backend stack (containers + network + volumes + orphans)...
REM -v removes named volumes (postgres_data, redis_data) => truly clean
REM --remove-orphans removes any container not in the current compose file
docker compose down -v --remove-orphans
if errorlevel 1 (
  echo   WARNING: docker compose down reported an error. Forcing container removal...
  for /f "tokens=*" %%c in ('docker compose ps -aq 2^>nul') do docker rm -f %%c >nul 2>&1
) else (
  echo   Backend fully removed.
)

:done
echo.
echo ============================================
echo   ALL STOPPED - CLEAN
echo ============================================
echo Removed: containers, network, and data volumes (postgres_data, redis_data).
echo Next start.bat will re-run migrations on a fresh database.
echo.
pause
endlocal
