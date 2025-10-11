#!/bin/bash

echo "========================================"
echo "  Inicializar Proveedores de Publicidad"
echo "========================================"
echo ""
echo "Este script creará los proveedores iniciales:"
echo "  - AdMob"
echo "  - Facebook Audience Network"
echo "  - Unity Ads"
echo "  - Personalizado"
echo ""
read -p "Presiona Enter para continuar o Ctrl+C para cancelar..."

echo ""
echo "Ejecutando script..."
echo ""

# Cambiar al directorio raíz del proyecto
cd "$(dirname "$0")/.."

# Ejecutar el script de Node.js
node scripts/init-providers.js

echo ""
echo "========================================"
echo "  Script completado"
echo "========================================"

