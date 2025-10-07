@echo off
echo ========================================
echo   Agregar campo 'enabled' a usuarios
echo ========================================
echo.
echo Este script agregara el campo 'enabled: false'
echo a todos los usuarios que no lo tengan.
echo.
echo Presiona Ctrl+C para cancelar o
pause

echo.
echo Ejecutando script...
echo.

cd /d "%~dp0\.."
node scripts/add-enabled-users.js

echo.
echo ========================================
echo   Script completado
echo ========================================
pause

