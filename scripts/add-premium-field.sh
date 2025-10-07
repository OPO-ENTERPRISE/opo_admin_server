#!/bin/bash

echo "========================================"
echo "  Agregar campo 'premium' a topics"
echo "========================================"
echo ""
echo "Este script agregará el campo 'premium: false'"
echo "a todos los topics que no lo tengan."
echo ""
read -p "Presiona Enter para continuar o Ctrl+C para cancelar..."

echo ""
echo "Ejecutando script..."
echo ""

# Cambiar al directorio raíz del proyecto
cd "$(dirname "$0")/.."

# Ejecutar el script de Node.js
node scripts/add-premium-field.js

echo ""
echo "========================================"
echo "  Script completado"
echo "========================================"

