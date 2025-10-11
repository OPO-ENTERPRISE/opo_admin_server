@echo off
echo ========================================
echo   Agregar campo 'type' a topics
echo ========================================
echo.
echo Este script agregara el campo 'type: topic'
echo a todos los topics que no lo tengan.
echo.
echo Presiona Ctrl+C para cancelar o
pause

echo.
echo Ejecutando script...
echo.

cd /d "%~dp0\.."
node scripts/add-type-field.js

echo.
echo ========================================
echo   Script completado
echo ========================================
pause

