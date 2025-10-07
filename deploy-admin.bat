@echo off
REM Script de despliegue de opo-admin-server en Google Cloud Run (Windows)

echo ==========================================
echo   Despliegue de opo-admin-server
echo   Google Cloud Run
echo ==========================================
echo.

REM Verificar que gcloud esté instalado
where gcloud >nul 2>nul
if errorlevel 1 (
    echo Error: gcloud CLI no esta instalado
    echo Instala desde: https://cloud.google.com/sdk/docs/install
    pause
    exit /b 1
)

echo Verificando configuracion de Google Cloud...
echo.

REM Obtener proyecto actual
for /f "delims=" %%i in ('gcloud config get-value project 2^>nul') do set PROJECT_ID=%%i

if "%PROJECT_ID%"=="" (
    echo Error: No hay proyecto de Google Cloud configurado
    echo Ejecuta: gcloud config set project TU_PROJECT_ID
    pause
    exit /b 1
)

echo Proyecto actual: %PROJECT_ID%
echo.

echo Este script desplegara opo-admin-server en Cloud Run
echo    - Region: europe-southwest1
echo    - Memoria: 512Mi
echo    - Instancias: 0-5 (autoscaling)
echo.
echo Presiona Ctrl+C para cancelar o
pause

echo.
echo Iniciando despliegue con Cloud Build...
echo.

REM Ejecutar Cloud Build
gcloud builds submit --config cloudbuild.yaml

if errorlevel 1 (
    echo.
    echo Error en el despliegue
    pause
    exit /b 1
)

echo.
echo ==========================================
echo   Despliegue completado
echo ==========================================
echo.
echo Para ver la URL del servicio:
echo    gcloud run services describe opo-admin-server --region=europe-southwest1 --format="value(status.url)"
echo.
echo Para ver los logs:
echo    gcloud run services logs read opo-admin-server --region=europe-southwest1 --limit=50
echo.
pause

