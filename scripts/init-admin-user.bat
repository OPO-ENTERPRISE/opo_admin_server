@echo off
REM Script para inicializar el usuario administrador en Windows
REM Uso: init-admin-user.bat <email> <password> <appId>

if "%~3"=="" (
    echo Uso: %0 ^<email^> ^<password^> ^<appId^>
    echo Ejemplo: %0 admin@example.com password123 1
    echo appId: 1=PN ^(Policía Nacional^), 2=PS ^(Policía Local/Guardia Civil^)
    exit /b 1
)

set EMAIL=%1
set PASSWORD=%2
set APPID=%3

echo 🔧 Inicializando usuario administrador...
echo 📧 Email: %EMAIL%
echo 🏢 App ID: %APPID%

REM Ejecutar el script Go
go run scripts/init-admin-user.go "%EMAIL%" "%PASSWORD%" "%APPID%"

echo ✅ Usuario administrador inicializado!
echo 🚀 Ahora puedes ejecutar el servidor con: go run cmd/admin/main.go
