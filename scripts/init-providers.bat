@echo off
echo ========================================
echo   Inicializar Proveedores de Publicidad
echo ========================================
echo.
echo Este script creara los proveedores iniciales:
echo   - AdMob
echo   - Facebook Audience Network
echo   - Unity Ads
echo   - Personalizado
echo.
echo Presiona Ctrl+C para cancelar o
pause

echo.
echo Ejecutando script...
echo.

cd /d "%~dp0\.."
node scripts/init-providers.js

echo.
echo ========================================
echo   Script completado
echo ========================================
pause

