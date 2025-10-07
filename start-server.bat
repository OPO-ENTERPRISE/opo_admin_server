@echo off
echo 🚀 Iniciando servidor de administración OPO...
echo.

REM Verificar que existe el archivo .env
if not exist .env (
    echo ⚠️  No se encontró archivo .env
    echo 📝 Copiando archivo de ejemplo...
    copy env.example .env
    echo.
    echo 🔧 Por favor, edita el archivo .env con tu configuración de MongoDB
    echo 📖 Consulta env.example para ver las variables necesarias
    echo.
    pause
    exit /b 1
)

echo ✅ Archivo .env encontrado
echo.

REM Verificar que Go está instalado
go version >nul 2>&1
if errorlevel 1 (
    echo ❌ Go no está instalado o no está en el PATH
    echo 📥 Descarga Go desde: https://golang.org/dl/
    pause
    exit /b 1
)

echo ✅ Go está instalado
echo.

REM Instalar dependencias
echo 📦 Instalando dependencias...
go mod tidy
if errorlevel 1 (
    echo ❌ Error instalando dependencias
    pause
    exit /b 1
)

echo ✅ Dependencias instaladas
echo.

REM Iniciar servidor
echo 🚀 Iniciando servidor en puerto 8081...
echo 📡 API Base Path: /api/v1
echo 🌐 URL: http://localhost:8081/api/v1/healthz
echo.
echo 💡 Para detener el servidor, presiona Ctrl+C
echo.

go run cmd/admin/main.go
