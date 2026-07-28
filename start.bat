@echo off
setlocal
cd /d "%~dp0"

echo ============================================
echo   Crypto Spot Signal - STARTING
echo ============================================

echo.
echo [1/3] Starting backend stack (docker)...
docker compose up -d postgres redis ai-service migrate backend
if errorlevel 1 (
  echo.
  echo ERROR: docker compose failed. Is Docker Desktop running?
  pause
  exit /b 1
)

echo.
echo [2/3] Waiting for backend health on http://localhost:8080/health ...
set /a tries=0
:waitloop
set /a tries+=1
powershell -NoProfile -Command "try { $r = Invoke-WebRequest -UseBasicParsing -TimeoutSec 3 http://localhost:8080/health; if ($r.StatusCode -eq 200) { exit 0 } else { exit 1 } } catch { exit 1 }" >nul 2>&1
if not errorlevel 1 goto healthy
if %tries% geq 30 (
  echo   Backend not healthy after 30 tries; continuing anyway.
  goto healthy
)
echo   ...still starting (%tries%/30)
timeout /t 2 >nul
goto waitloop

:healthy
echo   Backend is up.

echo.
echo [3/3] Starting frontend (npm run dev) in a new window...
if not exist "web\node_modules" (
  echo   Installing web dependencies first time...
  pushd web
  call npm install
  popd
)
start "crypto-spot-signal-web" cmd /k "cd /d "%~dp0web" && npm run dev"

echo.
echo ============================================
echo   READY
echo   Backend API : http://localhost:8080
echo   Frontend    : http://localhost:3000
echo ============================================
echo Frontend runs in the separate window titled "crypto-spot-signal-web".
echo Run stop.bat to shut everything down.
echo.
pause
endlocal
