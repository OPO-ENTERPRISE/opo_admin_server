@echo off
echo ========================================
echo   Agregar campo 'premium' a topics
echo ========================================
echo.
echo Este script agregara el campo 'premium: false'
echo a todos los topics que no lo tengan.
echo.
echo Presiona Ctrl+C para cancelar o
pause

echo.
echo Ejecutando script...
echo.

cd /d "%~dp0\.."
node scripts/add-premium-field.js

echo.
echo ========================================
echo   Script completado
echo ========================================
pause

