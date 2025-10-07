#!/bin/bash

echo "========================================"
echo "  Agregar campo 'enabled' a usuarios"
echo "========================================"
echo ""
echo "Este script agregará el campo 'enabled: false'"
echo "a todos los usuarios que no lo tengan."
echo ""
read -p "Presiona Enter para continuar o Ctrl+C para cancelar..."

echo ""
echo "Ejecutando script..."
echo ""

# Cambiar al directorio raíz del proyecto
cd "$(dirname "$0")/.."

# Ejecutar el script de Node.js
node scripts/add-enabled-users.js

echo ""
echo "========================================"
echo "  Script completado"
echo "========================================"

